/**
 * Copyright (c) 2018 Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/operator-framework/operator-lib/leader"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("leader")

// BecomeLeader with pre-check cluster status - is there a previous pod in bad state?
func BecomeLeader(ctx context.Context, cfg *rest.Config, lockName, namespace string) error {
	client, _ := k8sClient.New(cfg, k8sClient.Options{})

	err := precheckLeaderLock(ctx, client, lockName, namespace)
	if err != nil {
		log.Error(err, "error while pre-checking leader lock")
	}

	// pre-checks done, proceed with SDK-provided election procedure
	err = leader.Become(ctx, lockName)
	return err
}

func precheckLeaderLock(ctx context.Context, client k8sClient.Client, lockName, ns string) error {
	existingConfigMap, e := getConfigMapWithLock(ctx, client, lockName, ns)
	if existingConfigMap == nil || e != nil {
		return e
	}

	currentPod := os.Getenv("POD_NAME")
	if currentPod == "" {
		return fmt.Errorf("required env POD_NAME not set")
	}

	log.Info("current pod name", "pod", currentPod)

	for _, lockOwner := range existingConfigMap.GetOwnerReferences() {
		if lockOwner.Name == currentPod {
			log.Info("leader lock is owned by current pod")
			return nil
		}
		log.Info("leader lock owner", "kind", lockOwner.Kind, "name", lockOwner.Name)
		e := checkupLeaderPodStatus(ctx, client, lockOwner, existingConfigMap, ns)
		if e != nil {
			return e
		}
	}

	return nil
}

// checkupLeaderPodStatus checks if leader pod status is marked with VMware-specific reason 'ProviderFailed'
// then deletes lock and pod
func checkupLeaderPodStatus(ctx context.Context, client k8sClient.Client, leaderRef metav1.OwnerReference, existingLock *corev1.ConfigMap, ns string) error {
	if leaderRef.Kind != "Pod" {
		log.Info("existing lock references a non-pod object", "kind", leaderRef.Kind)
		return nil
	}

	leaderPod := &corev1.Pod{}
	err := client.Get(ctx, k8sClient.ObjectKey{Namespace: ns, Name: leaderRef.Name}, leaderPod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("leader pod not found", "pod", leaderRef.Name, "namespace", ns)
			return nil
		}
		log.Error(err, "error while reading leader pod")
		return err
	}

	log.Info("leader pod status", "phase", leaderPod.Status.Phase, "reason", leaderPod.Status.Reason)

	if leaderPod.Status.Reason == "ProviderFailed" {
		log.Info("deleting failed leader pod and lock ConfigMap to unblock leader election", "reason", leaderPod.Status.Reason)
		if err := deleteLeader(ctx, client, leaderPod, existingLock); err != nil {
			return err
		}
	}

	return nil
}

func getConfigMapWithLock(ctx context.Context, client k8sClient.Client, lockName, ns string) (*corev1.ConfigMap, error) {
	existingConfigMap := &corev1.ConfigMap{}
	e := client.Get(ctx, k8sClient.ObjectKey{Namespace: ns, Name: lockName}, existingConfigMap)
	if e != nil {
		if apierrors.IsNotFound(e) {
			log.Info("leader lock not found", "lock", lockName, "namespace", ns)
			return nil, nil
		}
		log.Error(e, "error while getting leader lock ConfigMap")
		return nil, e
	}
	return existingConfigMap, nil
}

// deleteLeader tries to delete pod and config map
func deleteLeader(ctx context.Context, client k8sClient.Client, leaderPod *corev1.Pod, configMapWithLock *corev1.ConfigMap) error {
	err := client.Delete(ctx, leaderPod)
	if err != nil {
		log.Error(err, "error deleting leader pod", "pod", leaderPod.Name)
		return err
	}

	err = client.Delete(ctx, configMapWithLock)
	switch {
	case apierrors.IsNotFound(err):
		log.Info("leader lock ConfigMap has already been deleted")
		return nil
	case err != nil:
		return err
	}

	return nil
}
