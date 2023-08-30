package error_middleware

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/segments/segments_core/segments_domain"
	"avito-internship-2023/internal/segments/segments_core/segments_services"
	"avito-internship-2023/internal/segments/segments_integrations/segments_dropbox"
	"avito-internship-2023/internal/segments/segments_repositories/segments_postgres"

	"github.com/gin-gonic/gin"
)

type ErrorType string

const (
	databaseError         ErrorType = "database_error"
	validationError       ErrorType = "validation_error"
	notFound              ErrorType = "not_found"
	parseError            ErrorType = "parse_error"
	externalResourceError ErrorType = "external_resource_error"
	unclassifiedError     ErrorType = "unclassified_error"
)

type internalError struct {
	Type ErrorType `json:"type"`
	Info string    `json:"info"`
}

func (err internalError) Error() string {
	return "Type: " + string(err.Type) + "; Info: " + err.Info
}

func New(logger common.Logger) gin.HandlerFunc {
	errorsMap := make(map[error]ErrorType)
	errorsMap[common.ErrBindFailed] = parseError
	errorsMap[common.ErrJSONUnmarshalling] = parseError
	errorsMap[common.ErrValidationFailed] = validationError
	errorsMap[sql.ErrNoRows] = notFound
	errorsMap[sql.ErrTxDone] = databaseError
	errorsMap[sql.ErrConnDone] = databaseError
	errorsMap[postgres.ErrInvalidContext] = unclassifiedError
	errorsMap[postgres.ErrInvalidValueType] = unclassifiedError
	errorsMap[segments_domain.ErrUserNotFound] = validationError
	errorsMap[segments_domain.ErrInvalidUserStatus] = validationError
	errorsMap[segments_services.ErrSegmentNotExist] = validationError
	errorsMap[segments_services.ErrSlugAlreadyInUse] = validationError
	errorsMap[segments_services.ErrUserDoesNotExist] = validationError
	errorsMap[segments_services.ErrTooMuchParameters] = validationError
	errorsMap[segments_dropbox.ErrUnexpectedBehaviour] = externalResourceError
	errorsMap[segments_postgres.ErrNoUsersToPick] = unclassifiedError

	codeMap := make(map[ErrorType]int)
	codeMap[parseError] = http.StatusBadRequest
	codeMap[validationError] = http.StatusBadRequest
	codeMap[notFound] = http.StatusBadRequest
	codeMap[databaseError] = http.StatusInternalServerError
	codeMap[externalResourceError] = http.StatusInternalServerError
	codeMap[unclassifiedError] = http.StatusInternalServerError

	return func(c *gin.Context) {
		buffToRequest := &bytes.Buffer{}
		buffToKeep := &bytes.Buffer{}
		multiWriter := io.MultiWriter(buffToRequest, buffToKeep)

		_, err := io.Copy(multiWriter, c.Request.Body)
		if err != nil {
			logger.Error("error when reading request body")
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		c.Request.Body = io.NopCloser(buffToRequest)

		// Waiting for all handlers to execute
		c.Next()

		if len(c.Errors) < 1 {
			return
		}

		outErrs := make([]internalError, 0, len(c.Errors))
		for _, processingError := range c.Errors {
			var internalType ErrorType
			found := false
			for customErrType, internalErrType := range errorsMap {
				if errors.Is(processingError, customErrType) {
					internalType = internalErrType
					found = true
					break
				}
			}
			if !found {
				internalType = unclassifiedError
			}

			outErr := internalError{Type: internalType, Info: processingError.Error()}

			outErrs = append(outErrs, outErr)
		}

		if buffToKeep.Len() > 1e5 {
			buffToKeep = &bytes.Buffer{}
			_, err = buffToKeep.WriteString("content consisting of more than 100_000 symbols is omitted in log")
			if err != nil {
				logger.Error("error when writing string to buffer")
				c.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		code, ok := codeMap[outErrs[0].Type]
		if !ok {
			code = http.StatusInternalServerError
		}

		jsonErrs := JSONErrors{Errors: outErrs}
		msg, mErr := json.Marshal(jsonErrs)
		if mErr != nil {
			panic(mErr)
		}

		logger.Errorw(string(msg),
			"Method", c.Request.Method,
			"URL", c.Request.URL,
			"Headers", c.Request.Header,
			"Body", buffToKeep.String(),
			"Cookies", c.Request.Cookies(),
			"Client IP", c.Request.RemoteAddr)

		c.JSON(code, jsonErrs)
	}
}
