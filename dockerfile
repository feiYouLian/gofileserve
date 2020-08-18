# 源镜像
FROM golang:latest
# 设置工作目录
WORKDIR /gofileserve
#将服务器的go工程代码加入到docker容器中
ADD . /gofileserve
# go设置env && 加载依赖
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.io,direct && go mod tidy
#go构建可执行文件
RUN go build .
# 创建挂载点
VOLUME ["/gofileserve/assets"]
# 暴露端口
EXPOSE 8080
# 最终运行docker的命令
ENTRYPOINT  ["./gofileserve"]
