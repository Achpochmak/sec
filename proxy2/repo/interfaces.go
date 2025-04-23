package repo

import "net/http"

type RequestSaver interface {
	Save(*http.Request) (string, error)
	Get(string) (*RequestData, error)
	GetEncoded(string) (*http.Request, error)
	List(int64) ([]*RequestData, error)
}

type ResponseSaver interface {
	Save(string, *http.Response) (string, error)
	Get(string) (*ResponseData, error)
	GetByRequest(string) (*ResponseData, error)
	List(int64) ([]*ResponseData, error)
}
