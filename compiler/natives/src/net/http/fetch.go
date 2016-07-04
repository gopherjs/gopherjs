// +build js

package http

import (
	"errors"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
)

// streamReader implements an io.ReadCloser wrapper for ReadableStream of https://fetch.spec.whatwg.org/.
type streamReader struct {
	pending []byte
	stream  *js.Object
}

func (r *streamReader) Read(p []byte) (n int, err error) {
	if len(r.pending) == 0 {
		var (
			bCh   = make(chan []byte)
			errCh = make(chan error)
		)
		r.stream.Call("read").Call("then",
			func(result *js.Object) {
				if result.Get("done").Bool() {
					errCh <- io.EOF
					return
				}
				bCh <- result.Get("value").Interface().([]byte)
			},
			func(reason *js.Object) {
				// Assumes it's a DOMException.
				errCh <- errors.New(reason.Get("message").String())
			},
		)
		select {
		case b := <-bCh:
			r.pending = b
		case err := <-errCh:
			return 0, err
		}
	}
	n = copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *streamReader) Close() error {
	// TOOD: Cannot do this because it's a blocking call, and Close() is often called
	//       via `defer resp.Body.Close()`, but GopherJS currently has an issue with supporting that.
	//       See https://github.com/gopherjs/gopherjs/issues/381 and https://github.com/gopherjs/gopherjs/issues/426.
	/*ch := make(chan error)
	r.stream.Call("cancel").Call("then",
		func(result *js.Object) {
			if result != js.Undefined {
				ch <- errors.New(result.String()) // TODO: Verify this works, it probably doesn't and should be rewritten as result.Get("message").String() or something.
				return
			}
			ch <- nil
		},
	)
	return <-ch*/
	r.stream.Call("cancel")
	return nil
}

// fetchTransport is a RoundTripper that is implemented using Fetch API. It supports streaming
// response bodies.
type fetchTransport struct{}

func (t *fetchTransport) RoundTrip(req *Request) (*Response, error) {
	headers := js.Global.Get("Headers").New()
	for key, values := range req.Header {
		for _, value := range values {
			headers.Call("append", key, value)
		}
	}
	opt := map[string]interface{}{
		"method":      req.Method,
		"headers":     headers,
		"credentials": "same-origin",
	}
	if req.Body != nil {
		// TODO: Find out if request body can be streamed into the fetch request rather than in advance here.
		//       See BufferSource at https://fetch.spec.whatwg.org/#body-mixin.
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			req.Body.Close() // RoundTrip must always close the body, including on errors.
			return nil, err
		}
		req.Body.Close()
		opt["body"] = body
	}
	respPromise := js.Global.Call("fetch", req.URL.String(), opt)

	var (
		respCh = make(chan *Response)
		errCh  = make(chan error)
	)
	respPromise.Call("then",
		func(result *js.Object) {
			header := Header{}
			result.Get("headers").Call("forEach", func(value, key *js.Object) {
				ck := CanonicalHeaderKey(key.String())
				header[ck] = append(header[ck], value.String())
			})

			contentLength := int64(-1)
			if cl, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64); err == nil {
				contentLength = cl
			}

			select {
			case respCh <- &Response{
				Status:        result.Get("status").String() + " " + StatusText(result.Get("status").Int()),
				StatusCode:    result.Get("status").Int(),
				Header:        header,
				ContentLength: contentLength,
				Body:          &streamReader{stream: result.Get("body").Call("getReader")},
				Request:       req,
			}:
			case <-req.Cancel:
			}
		},
		func(reason *js.Object) {
			select {
			case errCh <- errors.New("net/http: fetch() failed"):
			case <-req.Cancel:
			}
		},
	)
	select {
	case <-req.Cancel:
		// TODO: Abort request if possible using Fetch API.
		return nil, errors.New("net/http: request canceled")
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}
