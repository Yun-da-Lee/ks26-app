#!/usr/bin/env bash
# 建置並推送映像檔到 Docker Hub。
# 帳號、映像檔名稱、目標平台全部走環境變數，腳本裡不寫死任何帳號。
#
#   export DOCKERHUB_USER=你的帳號
#   ./build.sh
set -euo pipefail

: "${DOCKERHUB_USER:?請先 export DOCKERHUB_USER=<你的 Docker Hub 帳號>}"

IMAGE_NAME="${IMAGE_NAME:-ks26-app}"
# 標籤用 commit SHA，保證唯一；絕不用 latest，否則節點快取會讓你跑到舊版。
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short=7 HEAD)}"
# 目標平台依叢集節點架構決定，回自己的環境請改這個預設值。
PLATFORM="${PLATFORM:-linux/amd64}"

# 明寫 registry，不吃 docker CLI 的隱含預設值。
IMAGE="docker.io/${DOCKERHUB_USER}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "building ${IMAGE} for ${PLATFORM}"

docker buildx build \
  --platform "${PLATFORM}" \
  --tag "${IMAGE}" \
  --push \
  .

echo
echo "pushed: ${IMAGE}"
echo "把這個標籤填進 deploy/deployment.yaml 的 image 與 configmap 的 IMAGE_TAG。"
