#!/usr/bin/env bash

exec codex exec "Review plan file placed at ${PLAN_WORKING_FILE}. Leave review result as ${INTERMEDIATE_DIR}/${ITERATION}_REVIEW.md. \
Make sure only write in that file, no other file is created."
