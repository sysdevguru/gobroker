package apex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"time"

	"github.com/alpacahq/apex/encryption"
	"github.com/valyala/fasthttp"
)

var snapPath = "/snap/api/v1/"

type SnapTag string

const (
	ID_DOCUMENT SnapTag = "ID_DOCUMENT"
	DOCUMENT    SnapTag = "DOCUMENT"
	OTHER       SnapTag = "OTHER"
)

type PostSnapResponse struct {
	ID *string `json:"id"`
}

type GetSnapMetadataResponse struct {
	ID            *string  `json:"id"`
	Account       *string  `json:"account"`
	Correspondent *string  `json:"correspondent"`
	Tag           *string  `json:"tag"`
	Tags          []string `json:"tags"`
	ImageName     *string  `json:"imageName"`
	Description   *string  `json:"description"`
	TakenOn       *string  `json:"takenOn"`
	UploadedBy    *struct {
		Subject    *string `json:"subject"`
		UserEntity *string `json:"userEntity"`
		UserClass  *string `json:"userClass"`
	} `json:"uploadedBy"`
	UploadedOn *string `json:"uploadedOn"`
}

func (a *Apex) PostSnap(image []byte, name string, tag SnapTag) (*string, error) {
	var id string
	if os.Getenv("BROKER_MODE") != "DEV" {
		req := fasthttp.AcquireRequest()
		req.SetRequestURI(
			fmt.Sprintf(
				"%v%vimages",
				os.Getenv("APEX_URL"),
				snapPath,
			))
		req.Header.SetMethod("POST")
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.SetBoundary("ALPACA")
		part, err := w.CreateFormFile("file", "image")
		if err != nil {
			return nil, err
		}
		_, err = part.Write(image)
		if err != nil {
			return nil, err
		}
		// ID_DOCUMENT for now, as we are using only ID_DOCUMENT right now.
		code := os.Getenv("APEX_CORRESPONDENT_CODE")
		tag := string(tag)
		metadata, err := json.Marshal(GetSnapMetadataResponse{
			Correspondent: &code,
			Tag:           &tag,
			ImageName:     &name,
		})
		if err != nil {
			return nil, err
		}
		w.WriteField("metadata", string(metadata))

		req.SetBody(body.Bytes())
		req.Header.SetContentType(w.FormDataContentType())
		resp := fasthttp.AcquireResponse()

		// it's slow
		a.request(req, resp, time.Minute)

		if resp.StatusCode() != fasthttp.StatusOK {
			return nil, fmt.Errorf(
				"failed to POST image to snap (code: %v response: %v)",
				resp.StatusCode(), resp.String())
		}
		m := PostSnapResponse{}
		err = json.Unmarshal(resp.Body(), &m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal snap upload response (%v)", err)
		}
		id = *m.ID
	} else {
		id = encryption.GenRandomKey(20)
	}
	return &id, nil
}

func (a *Apex) GetSnap(id string) (*string, error) {
	uri := fmt.Sprintf(
		"%v%vimages/%v",
		os.Getenv("APEX_URL"),
		snapPath,
		id,
	)
	var body []byte
	// snap is slow, so let's set a long timeout
	if _, err := a.call(uri, "GET", nil, &body, time.Minute); err != nil {
		return nil, err
	}
	image := base64.StdEncoding.EncodeToString(body)
	return &image, nil
}

func (a *Apex) GetSnapMetadata(id string) (*GetSnapMetadataResponse, error) {
	uri := fmt.Sprintf(
		"%v%vimages/%v/metadata",
		os.Getenv("APEX_URL"),
		snapPath,
		id,
	)
	m := GetSnapMetadataResponse{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
