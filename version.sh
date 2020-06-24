
#!/bin/bash
ARG=$1

APP_NAME=s3cf

if [ "$DOCKER_ID" == "" ]; then
  DOCKER_ID=tonymackay
fi

if [ "$PASSWORD" == "" ]; then
  PASSWORD=$(cat ~/.docker/password.txt)
fi

VERSION=$(git describe)
if [[ $VERSION != v* ]]; then
  VERSION="dev-$(git log -n1 --format=format:"%H")"
fi

if [ "$ARG" == "build" ]; then
  echo "build Docker image"
  docker build -t "$DOCKER_ID/$APP_NAME" --build-arg VERSION=$VERSION .
  echo "create tag $VERSION"
  docker tag "$DOCKER_ID/$APP_NAME" "$DOCKER_ID/$APP_NAME:$VERSION"
fi

if [ "$ARG" == "publish" ]; then
  echo "publish Docker image"
  echo "$PASSWORD" | docker login -u "$DOCKER_ID" --password-stdin
  docker push "$DOCKER_ID/$APP_NAME:$VERSION"
fi

if [ "$ARG" == "test" ]; then
  echo $DOCKER_ID
  echo $APP_NAME
  echo $VERSION
fi