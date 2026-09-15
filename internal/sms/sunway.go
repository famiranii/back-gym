package sms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultEndpoint = "https://payamakafzar.ir/smsws/HttpService.ashx"

type Client struct {
	endpoint   string
	username   string
	password   string
	from       string
	httpClient *http.Client
}

type Option func(*Client)

func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func NewClient(
	username string,
	password string,
	from string,
	opts ...Option,
) *Client {
	client := &Client{
		endpoint: defaultEndpoint,
		username: username,
		password: password,
		from:     from,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

type SendResult struct {
	Recipient string
	MessageID int64
}

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"sunwaysms: error %d: %s",
		e.Code,
		e.Message,
	)
}

var errorMessages = map[int]string{
	51:  "wrong username or password",
	52:  "empty username or password",
	53:  "recipient count exceeds limit",
	54:  "recipient number empty",
	55:  "recipient number invalid",
	59:  "message body empty",
	60:  "server busy",
	61:  "sender number invalid",
	62:  "sender number empty",
	63:  "ip not authorized",
	66:  "checking message id count does not match recipient count",
	67:  "checking message id array exceeds limit",
	68:  "checking message id empty",
	69:  "checking message id invalid",
	70:  "user deactivated",
	77:  "not a web service user",
	78:  "not an sms management panel user",
	80:  "web service disabled",
	201: "recipient number format wrong",
	202: "operator of recipient number unknown",
	203: "insufficient credit",
	206: "invalid operator number",
	300: "sms containing links not permitted",
	400: "request count exceeds allowed limit",
	666: "service temporarily disabled",
	777: "ip blocked",
	888: "sender number not authenticated",
	999: "sms not permitted",
}

func (c *Client) Send(
	ctx context.Context,
	recipients []string,
	message string,
	flash bool,
) ([]SendResult, error) {
	if len(recipients) == 0 {
		return nil, fmt.Errorf("sunwaysms: no recipients")
	}

	if len(recipients) > 1000 {
		return nil, fmt.Errorf(
			"sunwaysms: too many recipients: %d",
			len(recipients),
		)
	}

	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("sunwaysms: empty message")
	}

	q := url.Values{}

	q.Set("service", "SendArray")
	q.Set("UserName", c.username)
	q.Set("Password", c.password)
	q.Set("To", strings.Join(recipients, ","))
	q.Set("Message", message)
	q.Set("From", c.from)
	q.Set("Flash", strconv.FormatBool(flash))

	reqURL := c.endpoint + "?" + q.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"sunwaysms: build request: %w",
			err,
		)
	}
	fmt.Printf(
		"SUNWAY endpoint=%s username=%q password=%q from=%q to=%q\n",
		c.endpoint,
		c.username,
		c.password,
		c.from,
		strings.Join(recipients, ","),
	)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"sunwaysms: request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"sunwaysms: unexpected http status: %d",
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"sunwaysms: read response: %w",
			err,
		)
	}

	return parseResponse(
		string(body),
		recipients,
	)
}

func parseResponse(
	body string,
	recipients []string,
) ([]SendResult, error) {
	body = strings.TrimSpace(body)

	if body == "" {
		return nil, fmt.Errorf(
			"sunwaysms: empty response",
		)
	}

	parts := strings.Split(body, ",")

	results := make([]SendResult, 0, len(parts))

	for i, part := range parts {
		value := strings.TrimSpace(part)

		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"sunwaysms: invalid response %q",
				body,
			)
		}

		if id <= 1000 {
			return nil, &APIError{
				Code:    int(id),
				Message: errorMessage(int(id)),
			}
		}

		recipient := ""

		if i < len(recipients) {
			recipient = recipients[i]
		}

		results = append(results, SendResult{
			Recipient: recipient,
			MessageID: id,
		})
	}

	if len(results) != len(recipients) {
		return nil, fmt.Errorf(
			"sunwaysms: response count %d does not match recipient count %d",
			len(results),
			len(recipients),
		)
	}

	return results, nil
}

func errorMessage(code int) string {
	if message, ok := errorMessages[code]; ok {
		return message
	}

	return "unknown error"
}
