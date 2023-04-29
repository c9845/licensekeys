# Intro:
This document details how to install the License Key Server application. This application allows you to create and manage license keys for your app(s).


# Requirements:
- Any modern OS.
- Resources:
  - 1x CPU.
  - 1Gb networking. 
  - 1GB of memory.
  - 10GB of storage.
- Should be installed on its own dedicated server, with a strongly set firewall, and other forms of system hardening.
- Port 8007 available within the host server for the app to run on unless you use a non-standard port in your config file.
- Some form of proxy to handle terminating HTTPS and proxying connections from port 80/443 to the port this app is running on.
- DNS resolution to the server this app is installed on with a valid HTTPS certificate.


# Install In Short:
- Deploy a new server system.
- Get the app files and executable.
- Deploy the database.
- Enable the app to run automatically upon boot.
- Configure a proxy.
- Run the app.


# Installation Steps:
1. Deploy a new server system.
    - Server install, LTS advised, no GUI needed. Ubuntu LTS is recommended.
    - Clean install with all updates installed.
    - Install extra packages. Ex.: `haveged` and `fail2ban`.
    - Make sure `sshd` is secured and using public key only.
      - Modify the file `/etc/ssh/sshd_config`.
      - PermitRootLogin no.
      - PasswordAuthentication no
      - PublicKeyAuthentication yes
    - Firewall configured. Allow access to ports 80 and 443.
    - Set a static IP.
    - Setup public IP address, if needed.
    - Setup DNS resolution to the server and the static IP.
    - Automatic security updates enabled.
      - Modify the file `/etc/apt/apt.conf.d/50unattended-upgrades`.
      - Unattended-Upgrade::Remove-Unused-Dependencies  "true";
    - Set correct timezone.
    - Harden the system in any other ways.

1. Get the app files.
    - Get files from Github or elsewhere. Files should be downloaded as a zip archive and a checksum. 
    - Confirm the checksum matches for the zip file.
        - For Ubuntu or other Linux systems, use the command `sha256sum`.
        - For Windows, use the powershell command `Get-FileHash`.
    - Unzip the zip file.
    - Copy the unzipped files and directories to a known clean location (nothing else in the directory).
    - Make the binary executable.
        - For Ubuntu or other Linux systems, use the `chmod +x /path/to/licensekeys`.

1. Deploy the database.
    - Run the binary.
    - The app will deploy the database automatically the first time it is run.
    - The app will also create a default config file. Do not delete this config file! Modify as needed.
    - If the database does not automatically deploy, deploy it using the `--deploy-db` flag.
    - The app will show that the database deployed successfully and the app is running.

1. Enable the app to run automatically.
    - This handles starting the app when the server boots.
    - See the "systemd-service" file in the documentation.
    - Copy and rename the "systemd-service.txt" file. Ex.: "license-keys.service".
    - Edit the "license-keys.service" file. Modify the ExecStart and User fields to the correct path and user.
    - Move the "license-keys.service" file to the `/etc/systemd/user` directory.
    - Enable the service by running `systemctl enable /etc/systemd/user/license-keys.service`.
    - Reboot the server to make sure the app starts when the server boots.

1. Configure a proxy.
    - This handles proxying traffic from ports 80/443 to the port the app serves on.
    - The proxy can also handle terminating the HTTPS connection.
    - If your proxy does not handle HTTPS certificate renewal, use certbot or another method.
    - Caddy or NGINX are recommended. 
    - Caddy:
        - Can automatically handle the HTTPS certificate renewal.
        - See caddyfile-sample.
    - NGINX:
        - Requires certbot for HTTPS certificate renewal.
        - See nginx-config-sample.
        - See nginx-logrotate-sample for preventing NGINX logs from getting too big.
        - See nginx-certbot-systemd-autorenew-service and nginx-certbot-systemd-autorenew-timer.

1. Run the app.
    - Run the binary if it isn't already running.
    - Confirm the app's logging does not show any errors.
    - Access the app via the URL you used when setting up DNS to make sure DNS, the proxy, HTTPS was configured properly.
    - Reboot the server to make sure automatic start of the app is functioning.
    - Log in using the default user, "admin@example.com", with the password being "admin@example.com".
    - Create a new user for yourself, change the default user's password, disable the default user, log out, and log in using your new user.
    - Configure the app's settings as needed.
    - Start using the app!
