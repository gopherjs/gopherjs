// +build js

package http

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"net/textproto"

	"github.com/gopherjs/gopherjs/js"
)

var DefaultTransport RoundTripper = &XhrTransport{
	inflight: map[*Request]*js.Object{},
}

type XhrTransport struct {
	inflight map[*Request]*js.Object
}

func (t *XhrTransport) RoundTrip(req *Request) (*Response, error) {
	xhrConstructor := js.Global.Get("XMLHttpRequest")
	if xhrConstructor == js.Undefined {
		return nil, errors.New("net/http: XMLHttpRequest not available")
	}
	xhr := xhrConstructor.New()
	xhr.Set("responseType", "arraybuffer")
	t.inflight[req] = xhr
	defer delete(t.inflight, req)

	respCh := make(chan *Response)
	errCh := make(chan error)

	xhr.Set("onload", func() {
		header, _ := textproto.NewReader(bufio.NewReader(bytes.NewReader([]byte(xhr.Call("getAllResponseHeaders").String() + "\n")))).ReadMIMEHeader()
		body := js.Global.Get("Uint8Array").New(xhr.Get("response")).Interface().([]byte)
		respCh <- &Response{
			Status:        xhr.Get("status").String() + " " + xhr.Get("statusText").String(),
			StatusCode:    xhr.Get("status").Int(),
			Header:        Header(header),
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
			Request:       req,
		}
	})

	xhr.Set("onerror", func(e *js.Object) {
		errCh <- errors.New("net/http: XMLHttpRequest failed")
	})

	xhr.Set("onabort", func(e *js.Object) {
		errCh <- errors.New("net/http: request canceled")
	})

	xhr.Call("open", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			xhr.Call("setRequestHeader", key, value)
		}
	}
	var body []byte
	if req.Body != nil {
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	xhr.Call("send", body)

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

func (t *XhrTransport) CancelRequest(req *Request) {
	t.inflight[req].Call("abort")
}
