<!-- markdownlint-disable MD013 -->
# What is persistent-volume-migrator?

The persistent-volume-migrator tool aims to help migrate Flex RBD Volumes to
[Ceph-CSI](https://github.com/ceph/ceph-csi) RBD Volumes.

Migration between Ceph-CSI Volumes is also supported.

> **Note** Migration of CephFS Volumes is not supported for now.

## Getting Started

**NOTE**: The procedure will come with downtime while the applications are
scaled down for the volume migration.

## Prerequisite

This guide assumes you have created a Rook-Ceph cluster with Flex and
[Ceph-CSI](https://github.com/ceph/ceph-csi).

1. We assume that you have healthy cluster i.e. `ceph status`
   shows `health: OK`
2. Minimum ceph version should be Ceph-CSI supported. Check
   [Ceph-CSI](https://github.com/ceph/ceph-csi#ceph-csi-features-and-available-versions)
   supported versions.
3. Stop the application pods that are consuming the flex volume(s) that need
   to be converted.
4. Ceph-CSI storageClass should be created in which you want to migrate.

## Steps to Migrate

1. Stop the application pods that are consuming the flex volume(s) that need
   to be converted.
2. Create migrator pod `kubectl create -f manifests/migrator.yaml`
3. Download the binary from release
   1. `wget https://github.com/ceph/persistent-volume-migrator/releases/download/v0.1.0-alpha/pv-migrator`
4. Run the command to [migrate the PVC(s)](#migrate-the-pvcs)

**NOTE**: source and destination StorageClass should use the same pool.

## Usage

### Migrate a Single PVC

```console
pv-migrator --pvc=<pvc-name> --pvc-namespace=<pvc-namespace>
   --destination-sc=<csi-storageclass-name-to-migrate>
   [--rook-namespace=rook-operator-namespace]
   [--ceph-cluster-namespace=ceph-cluster-namespace]
```

   1. `--pvc`: **required**: name of the pvc to migrate
   2. `--pvc-ns`: **required**: namespace of the PVC which is going to migrate
   3. `--destination-sc`: **required**: name of the storageclass in which you
      want mirgrate.
   4. `--rook-ns`: **optional** namespace where the rook operator is running.
      **default: rook-ceph**.
   5. `--ceph-cluster-ns`: **optional** namespace where the ceph cluster is
      running. **default: rook-ceph**.

Example:

```console
pv-migrator --pvc=rbd-pvc --pvc-ns=default --destination-sc=csi-rook-ceph-block
```

```console
I1125 07:56:22.247311      63 log.go:34] Create Kubernetes Client
I1125 07:56:22.259115      63 log.go:34] List all the PVC from the source storageclass
I1125 07:56:22.261205      63 log.go:34] 1 PVCs found with source StorageClass
I1125 07:56:22.261221      63 log.go:34] Start Migration of PVCs to CSI
I1125 07:56:22.261226      63 log.go:34] migrating PVC "rbd-pvc" from namespace "default"
I1125 07:56:22.261229      63 log.go:34] Fetch PV information from PVC rbd-pvc
I1125 07:56:22.266734      63 log.go:34] PV found "pvc-30a01887-8821-4baf-835c-16a7e55ba7f0"
---
I1125 07:56:26.483172      63 log.go:34] successfully renamed volume csi-vol-2f8de58f-4dc5-11ec-a130-0242ac110005 -> pvc-30a01887-8821-4baf-835c-16a7e55ba7f0
I1125 07:56:26.483194      63 log.go:34] Delete old PV object: pvc-30a01887-8821-4baf-835c-16a7e55ba7f0
I1125 07:56:26.504578      63 log.go:34] waiting for PV pvc-30a01887-8821-4baf-835c-16a7e55ba7f0 in state &PersistentVolumeStatus{Phase:Bound,Message:,Reason:,} to be deleted (0 seconds elapsed)
I1125 07:56:26.702921      63 log.go:34] deleted persistent volume pvc-30a01887-8821-4baf-835c-16a7e55ba7f0
I1125 07:56:26.702944      63 log.go:34] successfully migrated pvc rbd-pvc
I1125 07:56:26.703335      63 log.go:34] Successfully migrated all the PVCs to CSI
```

### Migrate All PVCs in a StorageClass

```console
pv-migrator --source-storageclass=<flex-vol-storageclass>
   --destination-sc=<csi-storageclass-name-to-migrate>
   [--rook-namespace=rook-operator-namespace]
   [--ceph-cluster-namespace=ceph-cluster-namespace]
```

   1. `--source-sc`: **required**: name of the storageclass in which you have
      your PVCs.
   2. `--destination-sc`: **required**: name of the storageclass to which you
      want to migrate.
   3. `--rook-ns`: **optional** namespace where the rook operator is running.
      **default: rook-ceph**.
   4. `--ceph-cluster-ns`: **optional** namespace where the ceph cluster
      is running. **default: rook-ceph**.

Example:

```console
pv-migrator --source-sc=rook-ceph-block --destination-sc=csi-rook-ceph-block
```

```console
I1125 07:56:14.760055      62 log.go:34] Create Kubernetes Client
I1125 07:56:14.774526      62 log.go:34] List all the PVC from the source storageclass
I1125 07:56:14.786113      62 log.go:34] 1 PVCs found with source StorageClass rook-ceph-block
I1125 07:56:14.786128      62 log.go:34] Start Migration of PVCs to CSI
I1125 07:56:14.786135      62 log.go:34] migrating PVC "rbd-pvc" from namespace "default"
I1125 07:56:14.786139      62 log.go:34] Fetch PV information from PVC rbd-pvc
I1125 07:56:14.789593      62 log.go:34] PV found "pvc-a62b1502-b1f4-403a-a0fb-7bf0464b9901"
---
I1125 07:56:17.569950      62 log.go:34] successfully renamed volume csi-vol-2a3f31ae-4dc5-11ec-8480-0242ac110005 -> pvc-a62b1502-b1f4-403a-a0fb-7bf0464b9901
I1125 07:56:17.569976      62 log.go:34] Delete old PV object: pvc-a62b1502-b1f4-403a-a0fb-7bf0464b9901
I1125 07:56:17.576867      62 log.go:34] waiting for PV pvc-a62b1502-b1f4-403a-a0fb-7bf0464b9901 in state &PersistentVolumeStatus{Phase:Bound,Message:,Reason:,} to be deleted (0 seconds elapsed)
I1125 07:56:17.773970      62 log.go:34] deleted persistent volume pvc-a62b1502-b1f4-403a-a0fb-7bf0464b9901
I1125 07:56:17.773992      62 log.go:34] successfully migrated pvc rbd-pvc
I1125 07:56:17.778567      62 log.go:34] Successfully migrated all the PVCs to CSI
```
