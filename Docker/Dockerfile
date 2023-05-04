
# 基础镜像
FROM centos:latest

# 设置工作目录
WORKDIR /app
RUN mkdir /app/logs/

# 复制应用程序和配置文件到容器中
COPY es_index /app/app
COPY config.yaml /app/config.yaml


RUN pwd

# 启动应用程序
CMD ["./app"]