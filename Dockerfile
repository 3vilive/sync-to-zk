FROM busybox

ENV ZK_SERVERS=localhost:2181
ENV SYNC_DIRS=sync_dir

COPY .build/sync-to-zk /bin/sync-to-zk
WORKDIR /sync-to-zk
ENTRYPOINT [ "/bin/sync-to-zk" ]
CMD [ ]
