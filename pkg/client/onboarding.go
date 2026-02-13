package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

// OnboardingScope provides operations on the first-run onboarding status.
type OnboardingScope struct {
	client *Client
}

// GetStatus returns the current onboarding status. The response structure is
// untyped and varies by Report Portal version, so it is returned as a generic map.
func (s *OnboardingScope) GetStatus(ctx context.Context, page *string) (map[string]any, error) {
	logger := s.client.logger.With(slog.String("operation", "GetOnboardingStatus"))
	logger.InfoContext(ctx, "API request")

	params := &api.OnBoardingUsingGETParams{
		Page: page,
	}

	response, err := s.client.raw.OnBoardingUsingGETWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "get onboarding status", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}
