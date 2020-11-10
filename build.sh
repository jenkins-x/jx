#!/bin/sh

if [ -z "$GCP_SA" ]
then
  echo "no GCP SA specified"
else
  echo "enabling GCP Service Account from $GCP_SA"
  gcloud auth activate-service-account --key-file $GCP_SA
fi


echo "building container image version: $VERSION"

gcloud builds submit --config cloudbuild.yaml --project jenkinsxio --substitutions=_VERSION="$VERSION"

