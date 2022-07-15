package errorhandling

import (
	"context"
	"database/sql"
	goErrors "errors"
	"runtime"

	"github.com/99designs/gqlgen/graphql"
	"github.com/neighborly/go-errors"
	"github.com/nrfta/go-log"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var (
	errorCodeMappings = map[errors.ErrorCode]string{
		errors.InternalError:    "INTERNAL_ERROR",
		errors.NotFound:         "NOT_FOUND",
		errors.InvalidArgument:  "INVALID_ARGUMENT",
		errors.Unauthenticated:  "UNAUTHENTICATED",
		errors.PermissionDenied: "PERMISSION_DENIED",
		errors.Unknown:          "UNKNOWN",
	}
)

type ErrorReporterFunc func(ctx context.Context, err error)

func ConfigureErrorPresenterFunc(reporterFunc ErrorReporterFunc) graphql.ErrorPresenterFunc {
	return func(ctx context.Context, e error) *gqlerror.Error {
		if _, ok := e.(*runtime.TypeAssertionError); ok {
			return graphql.DefaultErrorPresenter(ctx, e)
		}

		err := graphql.DefaultErrorPresenter(ctx, e)
		var customError error

		var gqlerr *gqlerror.Error
		if goErrors.As(e, &gqlerr) {
			customError = createCustomError(gqlerr.Unwrap())
		} else {
			customError = createCustomError(err)
		}

		if customError == nil {
			return err
		}
		reportAndLogError(reporterFunc, ctx, customError)

		err.Message = errors.DisplayMessage(customError)
		err.Extensions = map[string]interface{}{
			"code": transformToGraphqlErrorCode(errors.Code(customError)),
		}
		return err
	}
}

// ConfigureRecoverFunc will better handle panic errors and recover from it
func ConfigureRecoverFunc() graphql.RecoverFunc {
	return func(_ context.Context, errInterface interface{}) error {
		var err error
		switch e := errInterface.(type) {
		case error:
			err = e
		default:
			// skip the panic handler stack frames until the actual panic
			err = errors.Newf("%+v", e)
		}
		return err
	}
}

func transformToGraphqlErrorCode(code errors.ErrorCode) string {
	if mapping, ok := errorCodeMappings[code]; ok {
		return mapping
	}

	return string(code)
}

// reportAndLogError prints errors if the error is an internal error
func reportAndLogError(reportError ErrorReporterFunc, ctx context.Context, err error) {
	if errors.Code(err) == errors.InternalError {
		reportError(ctx, err)

		stack := errors.StackTrace(err)
		// only log errors that we don't control
		log.RequestLogger(ctx).Errorf("%s: %+v%+v", errors.InternalError, err.Error(), stack)
	}
}

// createCustomError: wrap error into custom error if not one already, otherwise return it.
func createCustomError(err error) error {
	if err == sql.ErrNoRows {
		return errors.WithDisplayMessage(
			errors.NotFound.Wrap(err, "record not found"),
			"Record Not Found",
		)
	}

	if errors.Code(err) != errors.InternalError {
		return err
	}

	// Handles gqlgen entity resolver wrapping the original error
	if unwrapped := goErrors.Unwrap(err); unwrapped != nil {
		return unwrapped
	}

	return err
}
