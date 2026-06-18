# tomato.beinai.cc Nginx SSL

This directory contains the Nginx virtual host config for `tomato.beinai.cc`.

## Local files

- Config: `deploy/nginx/tomato.beinai.cc.conf`
- Certificate bundle: `deploy/ssl/tomato.beinai.cc/tomato.beinai.cc_bundle.crt`
- Private key: `deploy/ssl/tomato.beinai.cc/tomato.beinai.cc.key`

The `deploy/ssl/` directory is ignored by Git because it contains private keys.

## Server install

Assuming the app is already running on the server with:

```text
127.0.0.1:18089 -> container port 8080
```

copy files to the server:

```bash
sudo mkdir -p /etc/nginx/ssl/tomato.beinai.cc
sudo cp tomato.beinai.cc_bundle.crt /etc/nginx/ssl/tomato.beinai.cc/
sudo cp tomato.beinai.cc.key /etc/nginx/ssl/tomato.beinai.cc/
sudo chmod 600 /etc/nginx/ssl/tomato.beinai.cc/tomato.beinai.cc.key
sudo chown -R root:root /etc/nginx/ssl/tomato.beinai.cc

sudo cp tomato.beinai.cc.conf /etc/nginx/conf.d/tomato.beinai.cc.conf
sudo nginx -t
sudo systemctl reload nginx
```

Open:

```text
https://tomato.beinai.cc
```
