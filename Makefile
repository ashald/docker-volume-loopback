VERSION             :=  0
AUTHOR              :=  ashald
PLUGIN_NAME			:=	docker-volume-loopback
PLUGIN_FULL_NAME	:=	${AUTHOR}/${PLUGIN_NAME}
ROOTFS_CONTAINER	:=	${PLUGIN_NAME}-rootfs
ROOTFS_IMAGE		:=	${AUTHOR}/${ROOTFS_CONTAINER}


build:
	GOOS=linux GOARCH=amd64 go build -o "$(PLUGIN_NAME)"


rootfs-image:
	docker build -t $(ROOTFS_IMAGE) .


rootfs: rootfs-image
	docker rm -vf $(ROOTFS_CONTAINER) || true
	docker create --name $(ROOTFS_CONTAINER) $(ROOTFS_IMAGE) || true
	mkdir -p plugin/rootfs
	rm -rf plugin/rootfs/*
	docker export $(ROOTFS_CONTAINER) | tar -x -C plugin/rootfs
	docker rm -vf $(ROOTFS_CONTAINER)


plugin: rootfs
	docker plugin disable $(PLUGIN_NAME) || true
	docker plugin rm --force $(PLUGIN_NAME) || true
	docker plugin create $(PLUGIN_NAME) ./plugin
	docker plugin enable $(PLUGIN_NAME)


plugin-push: rootfs
	docker plugin rm --force $(PLUGIN_FULL_NAME) || true
	docker plugin create $(PLUGIN_FULL_NAME) ./plugin
	docker plugin create $(PLUGIN_FULL_NAME):$(VERSION) ./plugin
	docker plugin push $(PLUGIN_FULL_NAME)
	docker plugin push $(PLUGIN_FULL_NAME):$(VERSION)


.PHONY: build rootfs-image rootfs plugin plugin-push
