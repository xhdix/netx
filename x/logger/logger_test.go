package logger

import (
	"io/ioutil"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/model"
)

func TestIntegration(t *testing.T) {
	log.SetHandler(discard.Default)
	client := httpx.NewClient(NewHandler(log.Log))
	client.ConfigureDNS("udp", "dns.google.com:53")
	resp, err := client.HTTPClient.Get("http://www.facebook.com")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil resp here")
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.HTTPClient.CloseIdleConnections()
}

func TestExtension(t *testing.T) {
	logger := NewHandler(log.Log)
	logger.OnMeasurement(model.Measurement{
		Extension: &model.ExtensionEvent{},
	})
}