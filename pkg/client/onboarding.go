package client

import (
	"context"
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

	params := &api.OnBoardingParams{
		Page: page,
	}

	response, err := s.client.raw.OnBoardingWithResponse(ctx, params)

	result, err := execute(ctx, logger, "get onboarding status", response, err)
	if err != nil {
		return nil, err
	}

	if *result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}
