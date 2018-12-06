package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	validatingwh "github.com/slok/kubewebhook/pkg/webhook/validating"
)

type config struct {
	certFile       string
	keyFile        string
	annoKeyRegex   string
	annoValueRegex string
	addr           string
}

func initFlags() *config {
	cfg := &config{}

	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	fl.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")
	fl.StringVar(&cfg.addr, "listen-addr", ":8080", "The address to start the server")
	fl.StringVar(&cfg.annoKeyRegex, "key", "", "The regex that matches the pod annotation key")
	fl.StringVar(&cfg.annoValueRegex, "value", "", "The regex that matches the pod annotation value")

	fl.Parse(os.Args[1:])
	return cfg
}

type podValidator struct {
	annoKeyRegex   *regexp.Regexp
	annoValueRegex *regexp.Regexp
	logger         log.Logger
}

func (v *podValidator) Validate(_ context.Context, obj metav1.Object) (bool, validatingwh.ValidatorResult, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return false, validatingwh.ValidatorResult{}, fmt.Errorf("not a pod")
	}

	// return early if no annotations found
	if pod.Annotations == nil {
		res := validatingwh.ValidatorResult{
			Valid:   false,
			Message: "no matching annotation found",
		}
		v.logger.Infof("pod %s is not valid", pod.Name)
		return false, res, nil
	}

	for annKey, annVal := range pod.Annotations {
		if v.annoKeyRegex.MatchString(annKey) {
			if v.annoValueRegex.MatchString(annVal) {
				v.logger.Infof("pod %s is valid", pod.Name)
				res := validatingwh.ValidatorResult{
					Valid:   true,
					Message: "pod is valid",
				}
				return false, res, nil
			}
		}
	}

	res := validatingwh.ValidatorResult{
		Valid:   false,
		Message: "no matching annotation found",
	}
	v.logger.Infof("pod %s is not valid", pod.Name)
	return false, res, nil

}

func main() {
	logger := &log.Std{Debug: true}

	cfg := initFlags()

	// Create our validator
	keyrgx, err := regexp.Compile(cfg.annoKeyRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid key regex: %s", err)
		os.Exit(1)
		return
	}

	valuergx, err := regexp.Compile(cfg.annoValueRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid value regex: %s", err)
		os.Exit(1)
		return
	}

	vl := &podValidator{
		annoKeyRegex:   keyrgx,
		annoValueRegex: valuergx,
		logger:         logger,
	}

	vcfg := validatingwh.WebhookConfig{
		Name: "podValidator",
		Obj:  &corev1.Pod{},
	}
	wh, err := validatingwh.NewWebhook(vcfg, vl, nil, nil, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	// Serve the webhook.
	logger.Infof("Listening on %s", cfg.addr)
	err = http.ListenAndServeTLS(cfg.addr, cfg.certFile, cfg.keyFile, whhttp.MustHandlerFor(wh))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
		os.Exit(1)
	}
}
