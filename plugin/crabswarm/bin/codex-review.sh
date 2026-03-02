#!/usr/bin/env bash

exec codex exec "Review plan file placed at ${PLAN_WORKING_FILE}. Leave review result to ${PLAN_REVIEW_FILE}. \
1) Make sure only write in that file, no other file is created.\
2) Don't focus on backward compatibility unless the plan explicitly refers to do so.\
3) Follow instruction in REVIEW_PERSPECTIVE.md if exists."

