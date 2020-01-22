package errorhandling

import (
	"context"
	"database/sql"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/vektah/gqlparser/gqlerror"

	"github.com/neighborly/go-errors"
	"github.com/nrfta/go-log"
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

func ConfigureErrorPresenterFunc(reporterFunc ErrorReporterFunc) handler.Option {
	return handler.ErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		e := createCustomError(err)
		reportAndLogError(reporterFunc, ctx, e)

		message := errors.DisplayMessage(e)
		code := errors.Code(e)

		// convert the error to a graphQL error
		err = &gqlerror.Error{
			Message: message,
			Extensions: map[string]interface{}{
				"code": transformToGraphqlErrorCode(code),
			},
		}

		return graphql.DefaultErrorPresenter(ctx, err)
	})
}

// ConfigureRecoverFunc will better handle panic errors and recover from it
func ConfigureRecoverFunc() handler.Option {
	return handler.RecoverFunc(func(ctx context.Context, errInterface interface{}) error {
		var err error
		switch e := errInterface.(type) {
		case error:
			err = e
		default:
			// skip the panic handler stack frames until the actual panic
			err = errors.Newf("%+v", e)
		}
		return err
	})
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

	return err
}
