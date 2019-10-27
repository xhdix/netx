// Package bodyreader contains the top HTTP body reader.
package bodyreader

import (
	"io"
	"net/http"
	"time"

	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
)

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(roundTripper http.RoundTripper) *Transport {
	return &Transport{roundTripper: roundTripper}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.roundTripper.RoundTrip(req)
	if err != nil {
		return
	}
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser: resp.Body,
		root:       model.ContextMeasurementRootOrDefault(req.Context()),
		tid:        transactionid.ContextTransactionID(req.Context()),
	}
	return
}

// CloseIdleConnections closes the idle connections.
func (t *Transport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}

type bodyWrapper struct {
	io.ReadCloser
	root *model.MeasurementRoot
	tid  int64
}

func (bw *bodyWrapper) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = bw.ReadCloser.Read(b)
	stop := time.Now()
	bw.root.Handler.OnMeasurement(model.Measurement{
		HTTPResponseBodyPart: &model.HTTPResponseBodyPartEvent{
			// "Read reads up to len(p) bytes into p. It returns the number of
			// bytes read (0 <= n <= len(p)) and any error encountered."
			Data:          b[:n],
			Duration:      stop.Sub(start),
			Error:         err,
			NumBytes:      int64(n),
			Time:          stop.Sub(bw.root.Beginning),
			TransactionID: bw.tid,
		},
	})
	return
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.root.Handler.OnMeasurement(model.Measurement{
		HTTPResponseDone: &model.HTTPResponseDoneEvent{
			Time:          time.Now().Sub(bw.root.Beginning),
			TransactionID: bw.tid,
		},
	})
	return
}
