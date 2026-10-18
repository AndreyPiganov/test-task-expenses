package currency

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/charmap"
)

const (
	defaultCBRURL      = "https://www.cbr.ru/scripts/XML_daily.asp"
	maxCBRResponseSize = 2 << 20
	requestTimeout     = 10 * time.Second
)

var ErrRateNotFound = errors.New("currency rate not found")

// Conversion is an amount converted to Russian rubles using an official rate.
type Conversion struct {
	AmountRUB decimal.Decimal
	Rate      decimal.Decimal
	RateDate  time.Time
}

// Converter converts an amount in an ISO 4217 currency to Russian rubles.
type Converter interface {
	ToRubles(context.Context, decimal.Decimal, string, time.Time) (Conversion, error)
}

// CBRClient loads official daily currency rates from the Bank of Russia.
type CBRClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewCBRClient() *CBRClient {
	return newCBRClient(defaultCBRURL, &http.Client{Timeout: requestTimeout})
}

func newCBRClient(endpoint string, httpClient *http.Client) *CBRClient {
	return &CBRClient{endpoint: endpoint, httpClient: httpClient}
}

func (c *CBRClient) ToRubles(
	ctx context.Context,
	amount decimal.Decimal,
	currencyCode string,
	date time.Time,
) (Conversion, error) {
	currencyCode = strings.ToUpper(strings.TrimSpace(currencyCode))
	if currencyCode == "RUB" || currencyCode == "RUR" {
		return Conversion{
			AmountRUB: amount.Round(2),
			Rate:      decimal.NewFromInt(1),
			RateDate:  date,
		}, nil
	}
	if len(currencyCode) != 3 {
		return Conversion{}, fmt.Errorf("%w: invalid ISO code %q", ErrRateNotFound, currencyCode)
	}

	endpoint, err := url.Parse(c.endpoint)
	if err != nil {
		return Conversion{}, fmt.Errorf("parse Bank of Russia endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("date_req", date.Format("02/01/2006"))
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Conversion{}, fmt.Errorf("create Bank of Russia request: %w", err)
	}
	request.Header.Set("Accept", "application/xml, text/xml")
	request.Header.Set("User-Agent", "family-expenses/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Conversion{}, fmt.Errorf("request Bank of Russia rates: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return Conversion{}, fmt.Errorf("Bank of Russia returned status %d", response.StatusCode)
	}

	daily, err := decodeDailyRates(io.LimitReader(response.Body, maxCBRResponseSize))
	if err != nil {
		return Conversion{}, fmt.Errorf("decode Bank of Russia rates: %w", err)
	}

	for _, rate := range daily.Rates {
		if strings.EqualFold(strings.TrimSpace(rate.CharCode), currencyCode) {
			nominal, parseErr := decimal.NewFromString(strings.TrimSpace(rate.Nominal))
			if parseErr != nil || !nominal.IsPositive() {
				return Conversion{}, fmt.Errorf("decode %s nominal %q", currencyCode, rate.Nominal)
			}
			value, parseErr := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(rate.Value), ",", "."))
			if parseErr != nil || !value.IsPositive() {
				return Conversion{}, fmt.Errorf("decode %s rate %q", currencyCode, rate.Value)
			}

			unitRate := value.Div(nominal)
			rateDate := date
			if parsedDate, dateErr := time.ParseInLocation("02.01.2006", daily.Date, date.Location()); dateErr == nil {
				rateDate = parsedDate
			}

			return Conversion{
				AmountRUB: amount.Mul(unitRate).Round(2),
				Rate:      unitRate,
				RateDate:  rateDate,
			}, nil
		}
	}

	return Conversion{}, fmt.Errorf("%w: %s on %s", ErrRateNotFound, currencyCode, date.Format("2006-01-02"))
}

type dailyRates struct {
	Date  string      `xml:"Date,attr"`
	Rates []dailyRate `xml:"Valute"`
}

type dailyRate struct {
	CharCode string `xml:"CharCode"`
	Nominal  string `xml:"Nominal"`
	Value    string `xml:"Value"`
}

func decodeDailyRates(reader io.Reader) (dailyRates, error) {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(strings.TrimSpace(charset)) {
		case "windows-1251", "cp1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		case "utf-8", "utf8":
			return input, nil
		default:
			return nil, fmt.Errorf("unsupported XML charset %q", charset)
		}
	}

	var result dailyRates
	if err := decoder.Decode(&result); err != nil {
		return dailyRates{}, err
	}
	return result, nil
}
