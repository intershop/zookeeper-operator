/**
 * Copyright (c) 2018 Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package e2e

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/pravega/zookeeper-operator/api/v1beta1"
	zk_e2eutil "github.com/pravega/zookeeper-operator/pkg/test/e2e/e2eutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Current ZooKeeper image", func() {
	It("emits JSON logs", func() {
		imageTag := os.Getenv("ZK_TEST_IMAGE_TAG")
		if imageTag == "" {
			Skip("ZK_TEST_IMAGE_TAG is not set")
		}

		kubeClient, err := kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		cluster := zk_e2eutil.NewDefaultCluster(testNamespace)
		cluster.Spec.Image = api.ContainerImage{
			Repository: "pravega/zookeeper",
			Tag:        imageTag,
			PullPolicy: corev1.PullIfNotPresent,
		}
		cluster.WithDefaults()
		cluster.Spec.Persistence.VolumeReclaimPolicy = "Delete"
		cluster.Status.Init()

		zk, err := zk_e2eutil.CreateCluster(logger, k8sClient, cluster)
		Expect(err).NotTo(HaveOccurred())
		Expect(zk_e2eutil.WaitForClusterToBecomeReady(logger, k8sClient, zk, 3)).NotTo(HaveOccurred())

		pods, err := zk_e2eutil.GetPods(k8sClient, zk)
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())

		logOptions := &corev1.PodLogOptions{Container: "zookeeper", TailLines: pointerToInt64(100)}
		logs, err := kubeClient.CoreV1().Pods(testNamespace).GetLogs(pods.Items[0].Name, logOptions).Do(ctx).Raw()
		Expect(err).NotTo(HaveOccurred())

		foundJSONRecord := false
		scanner := bufio.NewScanner(strings.NewReader(string(logs)))
		for scanner.Scan() {
			var record map[string]interface{}
			if json.Unmarshal(scanner.Bytes(), &record) == nil && len(record) > 0 {
				foundJSONRecord = true
				break
			}
		}
		Expect(scanner.Err()).NotTo(HaveOccurred())
		Expect(foundJSONRecord).To(BeTrue())

		Expect(zk_e2eutil.DeleteCluster(logger, k8sClient, zk)).To(Succeed())
		Expect(zk_e2eutil.WaitForClusterToTerminate(logger, k8sClient, zk)).To(Succeed())
	})
})

func pointerToInt64(value int64) *int64 {
	return &value
}
