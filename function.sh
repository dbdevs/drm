#!/usr/bin/env bash

function drm()
{
  output="$(~/.drm/bin/drm "$@")"
  if [[ $? -ne 0 ]]; then
    echo "The command did not succeed. Please try again."
    return $?
  fi

  lines=("${(@f)$(echo $output)}")

  for line in $lines; do
    if [[ "$line" == "DRM_CONTAINER_NAME="* ]]; then
      export $line
    elif [[ "$line" == "DRM_IMAGE_NAME="* ]]; then
      export $line
    elif [[ "$line" == "DRM_FULL_IMAGE_NAME="* ]]; then
      export $line
    else
      echo $line
    fi
  done

  if [[ "$1" == "use" ]]; then
    function bundle()
    {
      drm run bundle "$@"
    }
    alias bundle='bundle'
  fi
  if [[ "$1" == "use" ]]; then
    function rake()
    {
      drm run rake "$@"
    }
    alias rake='rake'
  fi
  if [[ "$1" == "use" ]]; then
    function rspec()
    {
      drm run rspec "$@"
    }
    alias rspec='rspec'
  fi
  if [[ "$1" == "use" ]]; then
    function ruby()
    {
      drm run ruby "$@"
    }
    alias ruby='ruby'
  fi
}