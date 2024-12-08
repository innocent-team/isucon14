#! /bin/bash

cd `dirname $0`

echo "Current Host: $HOSTNAME"

if [[ "$HOSTNAME" == isucon1 ]]; then
  INSTANCE_NUM="1"
elif [[ "$HOSTNAME" == isucon2 ]]; then
  INSTANCE_NUM="2"
elif [[ "$HOSTNAME" == isucon3 ]]; then
  INSTANCE_NUM="3"
else
  echo "Invalid host"
  exit 1
fi

set -ex

git pull

sudo systemctl daemon-reload

# env
install -o isucon -g isucon -m 755 ./conf/env/${INSTANCE_NUM}/env.sh /home/isucon/env.sh

# NGINX
sudo install -o root -g root -m 644 ./conf/nginx/conf.d/isuride.conf /etc/nginx/conf.d/isuride.conf
sudo install -o root -g root -m 644 ./conf/nginx/nginx.conf /etc/nginx/nginx.conf

if [[ "$INSTANCE_NUM" == 2 ]]; then
  sudo nginx -t

  sudo systemctl restart nginx
  sudo systemctl enable nginx
  sudo systemctl enable --now isuride-matcher
else
  sudo systemctl disable --now nginx.service
  sudo systemctl disable --now isuride-matcher
fi

# otel-contrib
# credential があるので config ファイルは含めていない
if [[ "$INSTANCE_NUM" == 3 ]]; then
  sudo systemctl disable --now otelcol-contrib.service
else
  sudo systemctl enable --now otelcol-contrib.service
fi


# APP
sudo install -o root -g root -m 644 ./conf/systemd/system/isuride-go.service /etc/systemd/system/isuride-go.service

if [[ "$INSTANCE_NUM" == 1 ]]; then
  sudo systemctl daemon-reload

  pushd go
  go build -o isuride
  popd
  sudo systemctl restart isuride-go.service
  sudo systemctl enable --now isuride-go.service

  sleep 2

  sudo systemctl status isuride-go.service --no-pager
else
  sudo systemctl disable --now isuride-go.service
fi

# MYSQL
sudo install -o root -g root -m 644 ./conf/mysql/mysql.conf.d/mysqld.cnf /etc/mysql/mysql.conf.d/mysqld.cnf
sudo install -o root -g root -m 644 ./conf/systemd/system/mysql-digest.service /etc/systemd/system/mysql-digest.service

if [[ "$INSTANCE_NUM" == 3 ]]; then
  sudo systemctl daemon-reload

  echo "MySQL restart したいなら手動で sudo systemctl restart mysql やってね"
#  sudo systemctl restart mysql
  sudo systemctl enable --now mysql
  sudo systemctl disable --now mysql-digest
#  sudo systemctl restart --now mysql-digest
else
  sudo systemctl disable --now mysql.service
  sudo systemctl disable --now mysql-digest.service
fi
