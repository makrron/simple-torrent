package server

import (
	"crypto/tls"
	"net/http"
	"strings"
)

func init() {
	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

func (s *Server) fetchSearchConfig(confurl string) error {
	if !strings.HasPrefix(confurl, "http://") && !strings.HasPrefix(confurl, "https://") {
		log.Println("fetchSearchConfig: unconfigured, using default embedded providers", confurl)
		return nil
	}
	log.Println("fetchSearchConfig: loading search config from", confurl)
	if err := s.searchEngine.LoadFromURL(confurl); err != nil {
		log.Println("[fetchSearchConfig]", err)
		return err
	}
	log.Printf("Loaded new search providers from %s", confurl)
	return nil
}
