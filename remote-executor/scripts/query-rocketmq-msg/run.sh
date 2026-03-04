#!/usr/bin/env bash
# query-rocketmq-msg/run.sh
# 参数由 executor 注入为环境变量：PARAM_TOPIC, PARAM_MESSAGE_ID

set -euo pipefail

TOPIC="${PARAM_TOPIC}"
MSG_ID="${PARAM_MESSAGE_ID}"
NAMESRV="${ROCKETMQ_NAMESRV:-192.168.0.43:31430}"

docker run --rm --privileged --network host \
  --entrypoint bash apache/rocketmq:5.1.4 \
  -c "/home/rocketmq/rocketmq-5.1.4/bin/mqadmin queryMsgByUniqueKey \
      -n ${NAMESRV} \
      -t ${TOPIC} \
      -i ${MSG_ID} \
      && echo \
      && cat /tmp/rocketmq/msgbodys/${MSG_ID}"
