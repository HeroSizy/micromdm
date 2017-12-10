package apply

import (
	"context"
	"io"

	"github.com/go-kit/kit/endpoint"
	"github.com/micromdm/dep"
)

type Endpoints struct {
	ApplyDEPTokensEndpoint   endpoint.Endpoint
	DefineDEPProfileEndpoint endpoint.Endpoint
	AppUploadEndpoint        endpoint.Endpoint
}

func (e Endpoints) UploadApp(ctx context.Context, manifestName string, manifest io.Reader, pkgName string, pkg io.Reader) error {
	request := appUploadRequest{
		ManifestName: manifestName,
		ManifestFile: manifest,
		PKGFilename:  pkgName,
		PKGFile:      pkg,
	}
	resp, err := e.AppUploadEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return resp.(appUploadResponse).Err
}

func (e Endpoints) DefineDEPProfile(ctx context.Context, p *dep.Profile) (*dep.ProfileResponse, error) {
	request := depProfileRequest{Profile: p}
	resp, err := e.DefineDEPProfileEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	response := resp.(depProfileResponse)
	return response.ProfileResponse, response.Err
}

func (e Endpoints) ApplyDEPToken(ctx context.Context, P7MContent []byte) error {
	req := depTokensRequest{P7MContent: P7MContent}
	resp, err := e.ApplyDEPTokensEndpoint(ctx, req)
	if err != nil {
		return err
	}
	return resp.(depTokensResponse).Err
}

func MakeApplyDEPTokensEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(depTokensRequest)
		err = svc.ApplyDEPToken(ctx, req.P7MContent)
		return depTokensResponse{
			Err: err,
		}, nil
	}
}

func MakeDefineDEPProfile(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(depProfileRequest)
		resp, err := svc.DefineDEPProfile(ctx, req.Profile)
		return &depProfileResponse{
			ProfileResponse: resp,
			Err:             err,
		}, nil
	}
}

func MakeUploadAppEndpiont(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(appUploadRequest)
		err = svc.UploadApp(ctx, req.ManifestName, req.ManifestFile, req.PKGFilename, req.PKGFile)
		return &appUploadResponse{
			Err: err,
		}, nil
	}
}

type appUploadRequest struct {
	ManifestName string
	ManifestFile io.Reader

	PKGFilename string
	PKGFile     io.Reader
}

type appUploadResponse struct {
	Err error `json:"err,omitempty"`
}

func (r appUploadResponse) error() error { return r.Err }

type depTokensRequest struct {
	P7MContent []byte `json:"p7m_content"`
}

type depTokensResponse struct {
	Err error `json:"err,omitempty"`
}

func (r depTokensResponse) error() error { return r.Err }

type depProfileRequest struct{ *dep.Profile }
type depProfileResponse struct {
	*dep.ProfileResponse
	Err error `json:"err,omitempty"`
}

func (r *depProfileResponse) error() error { return r.Err }