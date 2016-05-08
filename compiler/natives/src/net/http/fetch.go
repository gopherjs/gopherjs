// +build js

package http

import (
	"errors"
	"io"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
)

// streamReader implements a wrapper for ReadableStreamDefaultReader of https://streams.spec.whatwg.org/.
type streamReader struct {
	pending []byte
	reader  *js.Object
}

func (r streamReader) Read(p []byte) (n int, err error) {
	if len(r.pending) == 0 {
		var (
			bCh   = make(chan []byte)
			errCh = make(chan error)
		)
		r.reader.Call("read").Call("then",
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

func (streamReader) Close() error {
	// TODO: Implement. Use r.reader.cancel(reason) maybe?
	return errors.New("not yet implemented")
}

// fetchTransport is a RoundTripper that is implemented using Fetch API. It supports streaming
// response bodies.
type fetchTransport struct{}

func (t *fetchTransport) RoundTrip(req *Request) (*Response, error) {
	headers := js.Global.Get("Headers").New()
	for key, values := range req.Header {
		for _, value := range values {
			headers.Call("set", key, value)
		}
	}
	respPromise := js.Global.Get("fetch").Invoke(req.URL.String(), map[string]interface{}{
		"method":  req.Method,
		"headers": headers,
	})

	var (
		respCh = make(chan *Response)
		errCh  = make(chan error)
	)
	respPromise.Call("then",
		func(result *js.Object) {
			// TODO: Decide which of these two to use. The latter is more reliable,
			//       but likely uses up slightly more performance. It seems some browsers either
			//       don't set statusText, or set it to something weird. For example, Chrome 50 (latest stable)
			//       doesn't set it. Latest Safari does set it to something like "HTTP 2.0 200" instead of the
			//       expected "OK". Firefox set it to the expected "OK. Not sure what's the future of statusText
			//       property, maybe it's deprecated and we shouldn't use it? It does not seem to be deprecated
			//       from a quick look at the docs, so maybe use it and hope the implementations are fixed soon?
			//statusText := result.Get("statusText").String()
			statusText := StatusText(result.Get("status").Int())

			// TODO: Make this better.
			header := Header{}
			result.Get("headers").Call("forEach", func(value, key *js.Object) {
				header[CanonicalHeaderKey(key.String())] = []string{value.String()} // TODO: Support multiple values.
			})

			// TODO: With streaming responses, this cannot be set.
			//       But it doesn't seem to be set even for non-streaming responses. In other words,
			//       this code is currently completely unexercised/untested. Need to test it. Probably
			//       by writing a http.Handler that explicitly sets Content-Type header? Figure this out.
			contentLength := int64(-1)
			if cl, err := strconv.ParseInt(result.Get("headers").Call("get", "content-length").String(), 10, 64); err == nil {
				contentLength = cl
			}

			respCh <- &Response{
				Status:        result.Get("status").String() + " " + statusText,
				StatusCode:    result.Get("status").Int(),
				Header:        header,
				ContentLength: contentLength,
				Body:          &streamReader{reader: result.Get("body").Call("getReader")},
				Request:       req,
			}
		},
		func(reason *js.Object) {
			// TODO: Better error.
			errCh <- errors.New("net/http: Fetch failed")
		},
	)
	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

// TODO: Implement?
/*func (t *fetchTransport) CancelRequest(req *Request) {
}*/
