/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"time"

	"github.com/Sirupsen/logrus"

	"k8s.io/test-infra/prow/jenkins"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/plank"
)

var (
	totURL   = flag.String("tot-url", "http://tot", "Tot URL")
	crierURL = flag.String("crier-url", "http://crier", "Crier URL")

	buildCluster = flag.String("build-cluster", "", "Path to file containing a YAML-marshalled kube.Cluster object. If empty, uses the local cluster.")

	jenkinsURL       = flag.String("jenkins-url", "http://jenkins-proxy", "Jenkins URL")
	jenkinsUserName  = flag.String("jenkins-user", "jenkins-trigger", "Jenkins username")
	jenkinsTokenFile = flag.String("jenkins-token-file", "/etc/jenkins/jenkins", "Path to the file containing the Jenkins API token.")
)

func main() {
	flag.Parse()

	logrus.SetFormatter(&logrus.JSONFormatter{})

	kc, err := kube.NewClientInCluster(kube.ProwNamespace)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting kube client.")
	}
	var pkc *kube.Client
	if *buildCluster == "" {
		pkc = kc.Namespace(kube.TestPodNamespace)
	} else {
		pkc, err = kube.NewClientFromFile(*buildCluster, kube.TestPodNamespace)
		if err != nil {
			logrus.WithError(err).Fatal("Error getting kube client to build cluster.")
		}
	}

	jenkinsSecretRaw, err := ioutil.ReadFile(*jenkinsTokenFile)
	if err != nil {
		logrus.WithError(err).Fatalf("Could not read token file.")
	}
	jenkinsToken := string(bytes.TrimSpace(jenkinsSecretRaw))

	jc := jenkins.NewClient(*jenkinsURL, *jenkinsUserName, jenkinsToken)

	c := plank.NewController(kc, pkc, jc, *crierURL, *totURL)
	for range time.Tick(30 * time.Second) {
		start := time.Now()
		if err := c.Sync(); err != nil {
			logrus.WithError(err).Error("Error syncing.")
		}
		logrus.Infof("Sync time: %v", time.Since(start))
	}
}
