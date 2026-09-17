// internal/handlers/place_handler.go

package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	placeSearchLimit  = 5
	maxSearchQuery    = 120
	mapIRSearchURL    = "https://map.ir/search/v2/autocomplete"
	defaultDailyLimit = 900
	placeCacheTTL     = 24 * time.Hour
	maxCachedSearches = 5000
)

type PlaceHandler struct {
	apiKey            string
	client            *http.Client
	dailyRequestLimit int

	mu            sync.Mutex
	cache         map[string]cachedPlaceSearch
	requestDay    string
	requestsToday int
}

type cachedPlaceSearch struct {
	results   []PlaceSearchResult
	expiresAt time.Time
}

func NewPlaceHandler(apiKey string, dailyRequestLimit int) *PlaceHandler {
	if dailyRequestLimit <= 0 {
		dailyRequestLimit = defaultDailyLimit
	}

	return &PlaceHandler{
		apiKey:            apiKey,
		dailyRequestLimit: dailyRequestLimit,
		cache:             make(map[string]cachedPlaceSearch),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type MapIRAutocompleteResponse struct {
	Value []MapIRPlace `json:"value"`
}

type MapIRPlace struct {
	Title   string `json:"title"`
	Address string `json:"address"`
	Geom    struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geom"`
}

type PlaceSearchResult struct {
	ID        int     `json:"id"`
	Title     string  `json:"title"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	DistanceM float64 `json:"distance_m"`
}

type PlaceSearchResponse struct {
	Addresses []PlaceSearchResult `json:"addresses"`
}

func (h *PlaceHandler) Search(c fiber.Ctx) error {
	if h.apiKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "place search service is not configured",
		})
	}

	query := strings.TrimSpace(c.Query("q"))

	if len([]rune(query)) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "search query must contain at least 3 characters",
		})
	}

	if len([]rune(query)) > maxSearchQuery {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "search query is too long",
		})
	}

	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid latitude",
		})
	}

	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid longitude",
		})
	}

	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "coordinates are out of range",
		})
	}

	cacheKey := placeCacheKey(query, lat, lng)
	if results, ok := h.cachedResults(cacheKey); ok {
		return c.JSON(PlaceSearchResponse{Addresses: results})
	}

	if !h.reserveMapIRRequest() {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "daily place search limit reached",
		})
	}

	params := url.Values{}
	params.Set("text", query)

	mapIRURL := mapIRSearchURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(
		c.Context(),
		http.MethodGet,
		mapIRURL,
		nil,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create search request",
		})
	}

	req.Header.Set("x-api-key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "place search service unavailable",
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf(
				"place search failed with status %d",
				resp.StatusCode,
			),
		})
	}

	var response MapIRAutocompleteResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "invalid place search response",
		})
	}

	results := make([]PlaceSearchResult, 0, len(response.Value))

	for index, place := range response.Value {
		if len(place.Geom.Coordinates) < 2 {
			continue
		}

		placeLng := place.Geom.Coordinates[0]
		placeLat := place.Geom.Coordinates[1]

		distance := distanceMeters(lat, lng, placeLat, placeLng)

		results = append(results, PlaceSearchResult{
			ID:        index,
			Title:     place.Title,
			Address:   place.Address,
			Latitude:  placeLat,
			Longitude: placeLng,
			DistanceM: distance,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceM < results[j].DistanceM
	})

	if len(results) > placeSearchLimit {
		results = results[:placeSearchLimit]
	}

	h.cacheResults(cacheKey, results)

	return c.JSON(PlaceSearchResponse{
		Addresses: results,
	})
}

func placeCacheKey(query string, lat, lng float64) string {
	normalizedQuery := strings.NewReplacer("ي", "ی", "ك", "ک").Replace(query)
	normalizedQuery = strings.Join(strings.Fields(strings.ToLower(normalizedQuery)), " ")

	return fmt.Sprintf("%s:%.3f:%.3f", normalizedQuery, lat, lng)
}

func (h *PlaceHandler) cachedResults(key string) ([]PlaceSearchResult, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cached, ok := h.cache[key]
	if !ok || time.Now().After(cached.expiresAt) {
		if ok {
			delete(h.cache, key)
		}
		return nil, false
	}

	return append([]PlaceSearchResult(nil), cached.results...), true
}

func (h *PlaceHandler) cacheResults(key string, results []PlaceSearchResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.cache) >= maxCachedSearches {
		h.cache = make(map[string]cachedPlaceSearch)
	}

	h.cache[key] = cachedPlaceSearch{
		results:   append([]PlaceSearchResult(nil), results...),
		expiresAt: time.Now().Add(placeCacheTTL),
	}
}

func (h *PlaceHandler) reserveMapIRRequest() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	tehran := time.FixedZone("Asia/Tehran", 3*60*60+30*60)
	today := time.Now().In(tehran).Format("2006-01-02")
	if h.requestDay != today {
		h.requestDay = today
		h.requestsToday = 0
	}

	if h.requestsToday >= h.dailyRequestLimit {
		return false
	}

	h.requestsToday++
	return true
}

func distanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*
			math.Cos(lat2Rad)*
			math.Sin(dLng/2)*
			math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
