package main

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"

	"github.com/bmutziu/function-add-k8s-labels-annotations/input/v1beta1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag", req.GetMeta().GetTag())

	// This creates a new response to the supplied request. Note that Functions
	// are run in a pipeline! Other Functions may have run before this one. If
	// they did, response.To will copy their desired state from req to rsp. Be
	// sure to pass through any desired state your Function is not concerned
	// with unmodified.
	rsp := response.To(req, response.DefaultTTL)

	// Input is supplied by the author of a Composition when they choose to run
	// your Function. Input is arbitrary, except that it must be a KRM-like
	// object. Supporting input is also optional - if you don't need to you can
	// delete this, and delete the input directory.
	in := &v1beta1.Input{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	// Mutate the labels and annotations
	in.Labels["xfn-owner"] += "-team"
	in.Labels["xfn-provisioned-by"] = "upbound-" + in.Labels["xfn-provisioned-by"]
	in.Annotations["added-by-xfn"] += "-euro"
	f.log.Info("Actual ", "labels: ", in.Labels, "1st", in.Labels["xfn-owner"])
	f.log.Info("Actual ", "annotations: ", in.Annotations)

	// TODO: Add your Function logic here!
	//
	// Take a look at function-sdk-go for some utilities for working with req
	// and rsp - https://pkg.go.dev/github.com/crossplane/function-sdk-go
	//
	// Also, be sure to look at the tips in README.md

	desired, err := request.GetDesiredComposedResources(req) // function-sdk-go
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired composed resources from %T", req))
		return rsp, nil
	}

	for name, dr := range desired {
		f.log.Debug("Desired Resource", "composed-resource-name", name)

		region, err := dr.Resource.GetString("spec.forProvider.region")
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot get values", req))
			return rsp, nil
		}
		f.log.Debug("Region ", "is: ", region)

		if name == "platformref-vpc" {
			tags, err := dr.Resource.GetStringObject("spec.forProvider.tags")
			if err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "cannot get values", req))
				return rsp, nil
			}
			f.log.Debug("Tags ", "are :", tags)
			for key, value := range tags {
				fmt.Println(key, "->", value)
			}
			f.log.Debug("Tags ", "for Owner", tags["Owner"])
		}

		meta.AddLabels(dr.Resource, in.Labels) // Crossplane-runtime helper
		meta.AddAnnotations(dr.Resource, in.Annotations)
	}

	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources from %T", req))
		return rsp, nil
	}

	response.Normalf(rsp, "Kubernetes Labels and Annotations Added Successfully")

	return rsp, nil
}
