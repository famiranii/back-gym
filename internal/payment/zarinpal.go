package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ZarinPalService struct {
	MerchantID  string
	RequestURL  string
	VerifyURL   string
	StartPayURL string
	CallbackURL string
	HTTPClient  *http.Client
}

func NewZarinPalService(
	merchantID string,
	requestURL string,
	verifyURL string,
	startPayURL string,
	callbackURL string,
) *ZarinPalService {
	return &ZarinPalService{
		MerchantID:  merchantID,
		RequestURL:  requestURL,
		VerifyURL:   verifyURL,
		StartPayURL: startPayURL,
		CallbackURL: callbackURL,

		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ------------------------------------------------------------
// Request
// ------------------------------------------------------------

type ZarinPalRequest struct {
	MerchantID  string        `json:"merchant_id"`
	Amount      int64         `json:"amount"`
	Currency    string        `json:"currency,omitempty"`
	Description string        `json:"description"`
	CallbackURL string        `json:"callback_url"`
	ReferrerID  string        `json:"referrer_id,omitempty"`
	Metadata    *ZarinPalMeta `json:"metadata,omitempty"`
}

type ZarinPalMeta struct {
	Mobile  string `json:"mobile,omitempty"`
	Email   string `json:"email,omitempty"`
	OrderID string `json:"order_id,omitempty"`
}

type ZarinPalRequestResponse struct {
	Data struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		Authority string `json:"authority"`
		FeeType   string `json:"fee_type"`
		Fee       int64  `json:"fee"`
	} `json:"data"`

	Errors []any `json:"errors"`
}

// RequestPayment creates a payment authority.
func (s *ZarinPalService) RequestPayment(
	ctx context.Context,
	amount int64,
	description string,
	orderID string,
	mobile string,
	email string,
) (string, error) {

	payload := ZarinPalRequest{
		MerchantID: s.MerchantID,
		Amount:     amount,

		// اگر سیستم قیمت پروژه‌ات ریال است این را حذف کن.
		// اگر طبق تنظیمات زرین‌پالت تومان ارسال می‌کنی:
		Currency: "IRT",

		Description: description,
		CallbackURL: s.CallbackURL,

		Metadata: &ZarinPalMeta{
			Mobile:  mobile,
			Email:   email,
			OrderID: orderID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal zarinpal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.RequestURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create zarinpal request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("zarinpal request failed: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var responseBody bytes.Buffer
		_, _ = responseBody.ReadFrom(res.Body)

		return "", fmt.Errorf(
			"zarinpal request returned http status %d: %s",
			res.StatusCode,
			responseBody.String(),
		)
	}

	var result ZarinPalRequestResponse

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(
			"decode zarinpal response: %w",
			err,
		)
	}

	if result.Data.Code != 100 {
		return "", fmt.Errorf(
			"zarinpal request failed: code=%d message=%s",
			result.Data.Code,
			result.Data.Message,
		)
	}

	if strings.TrimSpace(result.Data.Authority) == "" {
		return "", fmt.Errorf("zarinpal returned empty authority")
	}

	return result.Data.Authority, nil
}

// ------------------------------------------------------------
// Verify
// ------------------------------------------------------------

type ZarinPalVerifyRequest struct {
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Authority  string `json:"authority"`
}

type ZarinPalVerifyResponse struct {
	Data struct {
		Code     int    `json:"code"`
		Message  string `json:"message"`
		RefID    int64  `json:"ref_id"`
		CardHash string `json:"card_hash"`
		CardPan  string `json:"card_pan"`
		FeeType  string `json:"fee_type"`
		Fee      int64  `json:"fee"`
	} `json:"data"`

	Errors []any `json:"errors"`
}

func (s *ZarinPalService) VerifyPayment(
	ctx context.Context,
	amount int64,
	authority string,
) (*ZarinPalVerifyResponse, error) {

	payload := ZarinPalVerifyRequest{
		MerchantID: s.MerchantID,
		Amount:     amount,
		Authority:  authority,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal zarinpal verify request: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.VerifyURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create zarinpal verify request: %w",
			err,
		)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"zarinpal verify request failed: %w",
			err,
		)
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var responseBody bytes.Buffer
		_, _ = responseBody.ReadFrom(res.Body)

		return nil, fmt.Errorf(
			"zarinpal verify returned http status %d: %s",
			res.StatusCode,
			responseBody.String(),
		)
	}

	var result ZarinPalVerifyResponse

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(
			"decode zarinpal verify response: %w",
			err,
		)
	}

	return &result, nil
}

// ------------------------------------------------------------
// Start Pay URL
// ------------------------------------------------------------

func (s *ZarinPalService) PaymentURL(authority string) string {
	return strings.TrimRight(
		s.StartPayURL,
		"/",
	) + "/" + authority
}
