#!/bin/bash

# set -x

echo "Welcome to kube-forensics-worker!"

export timestamp=$(date +"%s")
short_container_id=${CONTAINER_ID:0:12}

# Default subpath to "forensics" if not provided
if [[ -z "${SUBPATH}" ]]; then
  export SUBPATH="forensics"
fi

echo Namespace: $NAMESPACE
echo Pod Name: $POD_NAME
echo Container ID: $CONTAINER_ID
echo Short Container ID: $short_container_id
echo Subpath: $SUBPATH
echo Destination Bucket: $DEST_BUCKET

local_dest_dir=/forensics/${SUBPATH}/${timestamp}
mkdir -p ${local_dest_dir}

kubectl describe pod ${POD_NAME} -n ${NAMESPACE} > ${local_dest_dir}/${POD_NAME}.txt
kubectl get pod ${POD_NAME} -n ${NAMESPACE} -o yaml > ${local_dest_dir}/${POD_NAME}.yaml

docker inspect ${CONTAINER_ID} > ${local_dest_dir}/${POD_NAME}-${short_container_id}-inspect.json
docker diff ${CONTAINER_ID} > ${local_dest_dir}/${POD_NAME}-${short_container_id}-diff.txt
docker export ${CONTAINER_ID} --output=${local_dest_dir}/${POD_NAME}-${short_container_id}-export.tar

for file in ${local_dest_dir}/*
do
    aws s3 cp ${file} ${DEST_BUCKET}/${SUBPATH}/${timestamp}_${NAMESPACE}_${POD_NAME}/ --acl bucket-owner-full-control
done

rm -rf ${local_dest_dir}
