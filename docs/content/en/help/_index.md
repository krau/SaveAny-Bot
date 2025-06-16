---
title: "Frequently Asked Questions"
weight: 15
---

# Frequently Asked Questions

## Upload to AList shows success but actually fails

Adjust the upload chunk size in the AList management page, and deploy AList in a more stable network environment to reduce the occurrence of this issue.

## Bot indicates successful download but files don't show up in AList

AList caches directory structures. Refer to the <a href="https://alist.nn.ci/guide/drivers/common.html#cache-expiration" target="_blank">documentation</a> to adjust cache expiration time.

## Docker deployment still can't connect to Telegram despite proxy configuration (client initialization timeout)

Docker cannot directly access the host network. If you're not familiar with its usage, please set the container to host mode.