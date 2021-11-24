#!/usr/bin/env bash
set -exEuo pipefail

: "${FUNCTION:=${1}}"

# Source https://github.com/rook/rook
use_local_disk() {
  sudo swapoff --all --verbose
  sudo umount /mnt
  # search for the device since it keeps changing between sda and sdb
  sudo wipefs --all --force /dev/"$(lsblk|awk '/14G/ {print $1}'| head -1)"1
  sudo lsblk
}

deploy_rook_ceph_with_flex() {
  wget https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/common.yaml
  < tests/manifests/modified_common.yaml tee -a common.yaml
  kubectl create -f common.yaml
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/crds.yaml
  wget https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/operator.yaml
  sed -i 's|ROOK_ENABLE_FLEX_DRIVER: "false"|ROOK_ENABLE_FLEX_DRIVER: "true"|g' operator.yaml
  sed -i 's|# - name: FLEXVOLUME_DIR_PATH|- name: FLEXVOLUME_DIR_PATH|g' operator.yaml
  sed -i 's|#   value: "<PathToFlexVolumes>"|  value: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"|g' operator.yaml
  kubectl create -f operator.yaml
  wget https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/cluster-test.yaml
  sed -i "s|#deviceFilter:|deviceFilter: $(lsblk|awk '/14G/ {print $1}'| head -1)|g" cluster-test.yaml
  kubectl create -f cluster-test.yaml
  kubectl create -f tests/manifests/migrator.yaml
  # wait_for_pod_to_be_ready_state check for osd pod to in ready state
  wait_for_osd_pod_to_be_ready_state
  # wait_for_pod_to_be_ready_state check for toolbox pod to in ready state
  wait_for_toolboxpod_to_be_ready_state
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/flex/storageclass-test.yaml
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/pvc.yaml
  # creating sample application pod, writing some data into pod and deletes the pod
  create_sample_pod_and_write_some_data_and_delete
  # creating csi resources sc
  create_csi_resources
}

create_csi_resources(){
  wget https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/storageclass-test.yaml
  sed -i "s|name: rook-ceph-block|name: csi-rook-ceph-block|g" storageclass-test.yaml
  set +e # adding +e, as creating SC below will give already exists error
  kubectl create -f storageclass-test.yaml
  set -e
}

create_sample_pod_and_write_some_data_and_delete(){
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/pod.yaml
  echo "this is sample file" > pod-sample-file.txt
  wait_for_sample_pod_to_be_ready_state
  kubectl cp pod-sample-file.txt csirbd-demo-pod:/var/lib/www/html
  kubectl delete -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/pod.yaml
}

test_flex_migration_for_all_pvc(){
  go build main.go
  MIGRATION_POD=$(kubectl -n rook-ceph get pod -l app=rook-ceph-migrator -o jsonpath='{.items[*].metadata.name}')
  kubectl -n rook-ceph cp main "$MIGRATION_POD":/root/
  kubectl -n rook-ceph exec -it "$MIGRATION_POD" -- sh -c "cd root/ && ./main --source-sc=rook-ceph-block --destination-sc=csi-rook-ceph-block"
  exit_code_of_last_command=$?
  if [ $exit_code_of_last_command -ne 0 ]; then
    echo "Exit code migration command is non-zero $exit_code_of_last_command. Migration failed"
    exit 1
  fi
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/pod.yaml
  wait_for_sample_pod_to_be_ready_state
  verify_file_data_and_file_data
}

test_flex_migration_for_single_pvc(){
  go build main.go
  MIGRATION_POD=$(kubectl -n rook-ceph get pod -l app=rook-ceph-migrator -o jsonpath='{.items[*].metadata.name}')
  kubectl -n rook-ceph cp main "$MIGRATION_POD":/root/
  kubectl -n rook-ceph exec -it "$MIGRATION_POD" -- sh -c "cd root/ && ./main --pvc=rbd-pvc --pvc-ns=default --destination-sc=csi-rook-ceph-block"
  kubectl create -f https://raw.githubusercontent.com/rook/rook/release-1.7/cluster/examples/kubernetes/ceph/csi/rbd/pod.yaml
  wait_for_sample_pod_to_be_ready_state
  verify_file_data_and_file_data
}

verify_file_data_and_file_data(){
  storage_class_name=$(kubectl get pvc rbd-pvc -o jsonpath='{.spec.storageClassName}')
  if [ "$storage_class_name" !=  "csi-rook-ceph-block" ]; then
    echo "Migration failed"
    exit 1
  fi
  pod_data="$(kubectl exec  -it csirbd-demo-pod -- sh -c "cat /var/lib/www/html/pod-sample-file.txt")"
  file_data=$(cat pod-sample-file.txt)
  echo "$pod_data"
  echo "$file_data"
  if [[ "$pod_data" != "$file_data" ]]; then
    echo "migration failed"
    exit 1
  fi
}

# wait_for_pod_to_be_ready_state check for osd pod to in ready state
wait_for_osd_pod_to_be_ready_state() {
  timeout 300 bash <<-'EOF'
    until [ $(kubectl get pod -l app=rook-ceph-osd -n rook-ceph -o jsonpath='{.items[*].metadata.name}' -o custom-columns=READY:status.containerStatuses[*].ready | grep -c true) -eq 1 ]; do
      echo "waiting for the OSD pods to be in ready state"
      sleep 1
    done
EOF
}

# wait_for_pod_to_be_ready_state check for osd pod to in ready state
wait_for_toolboxpod_to_be_ready_state() {
  timeout 200 bash <<-'EOF'
    until [ $(kubectl get pod -l app=rook-ceph-migrator -n rook-ceph -o jsonpath='{.items[*].metadata.name}' -o custom-columns=READY:status.containerStatuses[*].ready | grep -c true) -eq 1 ]; do
      echo "waiting for the toolbox pods to be in ready state"
      sleep 1
    done
EOF
}

wait_for_sample_pod_to_be_ready_state() {
  timeout 200 bash <<-'EOF'
    until [ $(kubectl get pod csirbd-demo-pod -o jsonpath='{.items[*].metadata.name}' -o custom-columns=READY:status.containerStatuses[*].ready | grep -c true) -eq 1 ]; do
      echo "waiting for the application pods to be in ready state"
      sleep 1
    done
EOF
}

########
# MAIN #
########

FUNCTION="$1"
shift # remove function arg now that we've recorded it
# call the function with the remainder of the user-provided args
if ! $FUNCTION "$@"; then
  echo "Call to $FUNCTION was not successful" >&2
  exit 1
fi
