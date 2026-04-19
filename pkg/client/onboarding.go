package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

// OnboardingScope provides operations on the first-run onboarding status.
type OnboardingScope struct {
	client *Client
}

// GetStatus returns the current onboarding status. The response structure is
// untyped and varies by Report Portal version (may be a JSON object or array),
// so it is returned as an untyped value.
func (s *OnboardingScope) GetStatus(ctx context.Context, page *string) (any, error) {
	logger := s.client.logger.With(slog.String("operation", "GetOnboardingStatus"))
	logger.InfoContext(ctx, "API request")

	params := &api.OnBoardingParams{
		Page: page,
	}

	resp, err := s.client.rawClient.OnBoarding(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: %w", err)
	}
	defer resp.Body.Close()

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if strings.Contains(resp.Header.Get("Content-Type"), "json") {
			return nil, fmt.Errorf("get onboarding status: %w", parseErrorBody(resp.StatusCode, body))
		}

		return nil, fmt.Errorf("get onboarding status: %w", newUnexpectedStatusError(resp.StatusCode, body))
	}

	var result any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("get onboarding status: %w", err)
		}
	}

	return result, nil
}
