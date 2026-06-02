package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

// OnboardingScope provides operations on the first-run onboarding status.
type OnboardingScope struct {
	scope core.Scope
}

// GetStatus returns the current onboarding status. The response structure is
// untyped and varies by Report Portal version (may be a JSON object or array),
// so it is returned as an untyped value.
func (s *OnboardingScope) GetStatus(ctx context.Context, page *string) (any, error) {
	logger := s.scope.Logger.With(slog.String("operation", "GetOnboardingStatus"))
	logger.InfoContext(ctx, "API request")

	params := &api.OnBoardingParams{
		Page: page,
	}

	resp, err := s.scope.RawClient.OnBoarding(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if strings.Contains(resp.Header.Get("Content-Type"), "json") {
			return nil, fmt.Errorf(
				"get onboarding status: %w",
				core.ParseErrorBody(resp.StatusCode, body),
			)
		}

		return nil, fmt.Errorf(
			"get onboarding status: %w",
			core.NewUnexpectedStatusError(resp.StatusCode, body),
		)
	}

	var result any
	if len(body) > 0 {
		err := json.Unmarshal(body, &result)
		if err != nil {
			return nil, fmt.Errorf("get onboarding status: %w", err)
		}
	}

	return result, nil
}

// NewOnboardingScope constructs a OnboardingScope for wiring from the facade.
func NewOnboardingScope(s core.Scope) *OnboardingScope {
	return &OnboardingScope{scope: s}
}
