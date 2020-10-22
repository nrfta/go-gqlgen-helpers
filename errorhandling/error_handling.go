package errorhandling

import (
	"context"
	"database/sql"
	"github.com/99designs/gqlgen/graphql"
	"github.com/neighborly/go-errors"
	"github.com/nrfta/go-log"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"strings"
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
	return func(ctx context.Context, err error) *gqlerror.Error {
		e := createCustomError(err)

		// HACK: errors from directives get wrapped in an unexported field. Only message gets copied to the outer error
		// so we encode info in the message and reify the errors.customError here.
		if gqlerr, ok := err.(*gqlerror.Error); ok {
			msgParts := strings.Split(gqlerr.Message, ";")
			if len(msgParts) == 2 {
				errCode := errors.ErrorCode(msgParts[0])
				if _, ok = errorCodeMappings[errCode]; ok {
					e = errors.WithDisplayMessage(errCode.New(gqlerr.Message), msgParts[1])
				}
			}
		}

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
	}
}

// ConfigureRecoverFunc will better handle panic errors and recover from it
func ConfigureRecoverFunc() graphql.RecoverFunc {
	return func(ctx context.Context, errInterface interface{}) error {
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

	return err
}
