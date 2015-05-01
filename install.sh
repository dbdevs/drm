#!/usr/bin/env bash

docker build -f Dockerfile_Mac -t drm_mac .
mkdir -p ~/.drm/bin
docker run --rm -v ~/.drm/bin:/.drm/bin -w /.drm/bin drm_mac
cp function.sh ~/.drm
chmod +x ~/.drm/bin/drm
if [ -e "${HOME}/.zshrc" ]; then
  echo "source ~/.drm/function.sh" >> ~/.zshrc
fi
if [ -e "${HOME}/.bashrc" ]; then
  echo "source ~/.drm/function.sh" >> ~/.bashrc
fi
docker rmi -f drm_mac