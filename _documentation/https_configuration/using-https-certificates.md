# Intro:
This document details how to use `certbot` to manage an HTTPS certificate and set up automatic renewal.


# Assumptions:
- You are using Cloudflare as your public DNS provider and have the ability to generate an API Token.
- You are using and knowledgeable in NGINX, `systemd`, LetsEncrypt, and `certbot`.
- You are using `systemd` as the system manager.
- You are using NGINX as your proxy and HTTPS terminator.
- You have `snap` (snapd) installed.


# Notes:
Setting up HTTPS is a multi-step process.  
1. Retrieve your initial certificate proving ownership of the domain this app will run on.
1. Configure NGINX to use the certificate and terminate the HTTPS connection.
1. Set up automatic renewal of the HTTPS certificate.

Two files are needed to handle automatic renewal. Each will need to be renamed and moved to a certain location.
- nginx-certbot-systemd-autorenew-service
- nginx-certbot-systemd-autorenew-timer


# Initial Certificate:
1. Make sure you do not have any previous installs of `certbot` or `certbot-auto` on your system. If so, remove them to reduce the chances of errors.

1. Make sure `snap` is up to date.
    - `sudo snap install core; sudo snap refresh core`

1. Install the certbot snap.
    - `sudo snap install --classic certbot`

1. Make sure cerbot is accessible correctly.
    - `sudo ln -s /snap/bin/certbot /usr/bin/certbot`

1. Install the certbot Cloudflare plugin.
    - `sudo snap install --classic certbot-dns-cloudflare`

1. Retrieve your Cloudflare API Token from the Cloudflare web portal. Use a token, not a global API key.
    - See https://certbot-dns-cloudflare.readthedocs.io/en/stable/ for more info on the API Token.

1. Create a credentials file to store the API Token.
    - `touch ~/certbot-cloudflare-credentials.ini`
    - Add the following line to the just-created file using your API Token.
      - `dns_cloudflare_api_token = replace-this-text-with-your-api-token`
    - Save the file.
    - Set the permission on the file.
      - `chmod 600 ~/certbot-cloudflare-credentials.ini`

1. Run the following `certbot` command to generate a certificate.
    - `certbot certonly --dns-cloudflare --dns-cloudflare-credentials ~/certbot-cloudflare-credentials.ini -d subdomain.example.com`
      - Where `subdomain.domain.com` is the URL you will access the app from.
      - Provide responses to questions as needed.
      - Wait for successful completion.

1. Update the paths in your NGINX configuration to match the paths to the generated certificate files.
   - See the notes in the nginx-config-sample file for more info.
   - This may be done automatically by `certbot`.

1. Restart NGINX and make sure your app is serving using the new certificate.
    - `nginx -s reload`


# Automatic Renewal:
1. Create a copy of the "nginx-certbot-systemd-autorenew-service" file renaming it as you see fit.
    - The file must end in ".service".
    - Move the copy to the `/etc/systemd/user` directory.

1. Read the file to understand what it is doing, however, you do not need to modify anything.

1. Create a copy of the "nginx-certbot-systemd-autorenew-timer" renaming it as needed.
    - The file must end in ".timer".
    - The filename must match the name of the ".service" file minus the different extension.
    - Move the copy to `/etc/systemd/user`.

1. Read the file to understand what it is doing, however, you do not need to modify anything.

1. Enable each file bu running the following commands:
    - `systemctl enable /path/to/my-service-file.service`.
    - `systemctl enable /path/to/my-timer-file.timer`.
    
1. Check logging output using `journalctl` and check that renewal happens. Typically renewal only happens when the existing certificate has less than 1 month until expiry. Using `cerbot --dry-run` and a short interval in your ".timer" file can be helpful for making sure renewal is functioning correctly.
