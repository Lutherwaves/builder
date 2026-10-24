#!/bin/sh
# Publishes a preview of this pull request.
curl -fsS -X POST -H "Authorization: Bearer $PREVIEW_DEPLOY_TOKEN" https://preview.example/deploy
