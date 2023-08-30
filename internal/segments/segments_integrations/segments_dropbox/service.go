package segments_dropbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"avito-internship-2023/internal/pkg/common"

	"github.com/google/uuid"
)

const (
	UploadUrl           = "https://content.dropboxapi.com/2/files/upload"
	GetTemporaryLinkUrl = "https://api.dropboxapi.com/2/files/get_temporary_link"
)

var (
	ErrUnexpectedBehaviour = errors.New("unexpected behavior")
)

type Service struct {
	ctx    context.Context
	logger common.Logger
	token  string
}

func NewService(ctx context.Context, logger common.Logger, token string) *Service {
	return &Service{ctx: ctx, logger: logger, token: token}
}

func (service *Service) SaveCSVReportWithURLAccess(content io.Reader) (string, error) {
	randString := uuid.New().String()[:4]
	path := "/report_" + time.Now().Format("2006-01-02T15:04:05") + "_" + randString + ".csv"

	err := service.saveFileWithPath(content, path)
	if err != nil {
		return "", err
	}

	url, err := service.getTempLink(path)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (service *Service) saveFileWithPath(content io.Reader, path string) error {
	execCtx, cancelExec := context.WithTimeout(service.ctx, 10*time.Second)
	defer cancelExec()

	req, err := http.NewRequestWithContext(execCtx, http.MethodPost, UploadUrl, content)
	if err != nil {
		return err
	}

	apiArgs := map[string]string{
		"path": path,
	}

	jsonApiArgs, err := json.Marshal(apiArgs)
	if err != nil {
		panic(err)
	}

	req.Header["Dropbox-API-Arg"] = []string{string(jsonApiArgs)}
	req.Header.Set("Authorization", "Bearer "+service.token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		service.logger.Error(ErrUnexpectedBehaviour, " status: ", resp.StatusCode, " body: ", string(body), " error: ", err)

		return fmt.Errorf("%w; %s", ErrUnexpectedBehaviour, "upload request to dropbox failed")
	}

	return nil
}

func (service *Service) getTempLink(path string) (string, error) {
	reqContent := map[string]string{
		"path": path,
	}

	reqBody, err := json.Marshal(reqContent)
	if err != nil {
		return "", err
	}

	execCtx, cancelExec := context.WithTimeout(service.ctx, 10*time.Second)
	defer cancelExec()

	req, err := http.NewRequestWithContext(execCtx, http.MethodPost, GetTemporaryLinkUrl, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+service.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		service.logger.Error(ErrUnexpectedBehaviour, " status: ", resp.StatusCode, " body: ", string(body), " error: ", err)

		return "", fmt.Errorf("%w; %s", ErrUnexpectedBehaviour, "get-link request to dropbox failed")
	}

	type getLinkResponse struct {
		Link string `json:"link"`
	}

	var decodedResponse getLinkResponse
	err = json.NewDecoder(resp.Body).Decode(&decodedResponse)
	if err != nil {
		return "", err
	}

	return decodedResponse.Link, nil
}
