// Package tlsdialer contains the TLS dialer
package tlsdialer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialer/connx"
	"github.com/ooni/netx/model"
)

// TLSDialer is the TLS dialer
type TLSDialer struct {
	ConnectTimeout      time.Duration // default: 30 second
	TLSHandshakeTimeout time.Duration // default: 10 second
	beginning           time.Time
	config              *tls.Config
	dialer              model.Dialer
	handler             model.Handler
	setDeadline         func(net.Conn, time.Time) error
}

// New creates a new TLS dialer
func New(
	beginning time.Time,
	handler model.Handler,
	dialer model.Dialer,
	config *tls.Config,
) *TLSDialer {
	return &TLSDialer{
		ConnectTimeout:      30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		beginning:           beginning,
		config:              config,
		dialer:              dialer,
		handler:             handler,
		setDeadline: func(conn net.Conn, t time.Time) error {
			return conn.SetDeadline(t)
		},
	}
}

// DialTLS dials a new TLS connection
func (d *TLSDialer) DialTLS(network, address string) (net.Conn, error) {
	ctx := context.Background()
	return d.DialTLSContext(ctx, network, address)
}

// DialTLSContext is like DialTLS, but with context
func (d *TLSDialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.ConnectTimeout)
	defer cancel()
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.config.Clone() // avoid polluting original config
	if config.ServerName == "" {
		config.ServerName = host
	}
	err = d.setDeadline(conn, time.Now().Add(d.TLSHandshakeTimeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	tlsconn := tls.Client(conn, config)
	start := time.Now()
	err = tlsconn.Handshake()
	stop := time.Now()
	var connID int64
	if mconn, ok := conn.(*connx.MeasuringConn); ok {
		connID = mconn.ID
	}
	m := model.Measurement{
		TLSHandshake: &model.TLSHandshakeEvent{
			ConnID: connID,
			Config: model.TLSConfig{
				NextProtos: config.NextProtos,
				ServerName: config.ServerName,
			},
			ConnectionState: newConnectionState(tlsconn.ConnectionState()),
			Duration:        stop.Sub(start),
			Error:           err,
			Time:            stop.Sub(d.beginning),
		},
	}
	conn.SetDeadline(time.Time{}) // clear deadline
	d.handler.OnMeasurement(m)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return tlsconn, err
}

func newConnectionState(s tls.ConnectionState) model.TLSConnectionState {
	return model.TLSConnectionState{
		CipherSuite:        s.CipherSuite,
		NegotiatedProtocol: s.NegotiatedProtocol,
		PeerCertificates:   simplifyCerts(s.PeerCertificates),
		Version:            s.Version,
	}
}

func simplifyCerts(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}