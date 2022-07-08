# Intro:
This document details how to install the License Key Server application. This application allows you to create and manage license keys for your app(s).


# Requirements:
- Ubuntu. LTS is advised. Other Linux OSes should also work.
- Minimum hardware:
  - 1x CPU, 
  - 1Gb networking, 
  - 1GB of memory, 
  - 10GB of storage.
- Should be installed on its own dedicated server, with a strongly set firewall, and other forms of system hardening.
- Firewall allowing inbound access to ports 80 and 443 for serving the app, handled via a proxy terminating the HTTPS connection.
- Port 8007 available within the host server for the app to run on unless you use a non-standard port in your config file.
- DNS resolution to the server this app is installed on with a valid HTTPS certificate.


# Install In Short:
- Deploy an Ubuntu server system.
- Install SQLite.
- Get the app files and executable.
- Deploy the database.
- Enable the app to run automatically upon boot.
- Install NGINX or another proxy and HTTPS terminator.
- Configure NGINX to serve this app on port 80/443.
- Configure an HTTPS certificate.


# Installation Steps:
1. Install Ubuntu.
    - Server install, LTS advised, no GUI needed.
    - Clean install with all updates installed.
    - Install extra packages. Ex.: `haveged` and `fail2ban`.
    - Make sure `sshd` is secured and using public key only.
      - Modify the file `/etc/ssh/sshd_config`.
      - PermitRootLogin no.
      - PasswordAuthentication no
      - PublicKeyAuthentication yes
    - Firewall configured. Access to ports 80 and 443 as needed.
    - Static IP set.
    - DNS resolution to the server and the static IP, public IP as needed.
    - Automatic security updates enabled.
      - Modify the file `/etc/apt/apt.conf.d/50unattended-upgrades`.
      - Unattended-Upgrade::Remove-Unused-Dependencies  "true";
    - Set correct timezone.

1. Get the app files.
    - Get files from Github or elsewhere. Files should be downloaded as a zip archive and a checksum. 
    - Confirm the checksum matches for the zip file.
        - For Ubuntu or other Linux systems, use the command `sha256sum`.
        - For Windows, use the powershell command `Get-FileHash`.
    - Extract the zip file.
    - Copy the extracted files and directories to a known clean location (nothing else in the directory).
    - Run `chmod` on the directory, if needed, to set your required permissions.
    - Make the binary executable using `chmod +x /path/to/binary`.
    - If you downloaded the souce code, not a built binary, download and install `golang` and build the binary.

1. Deploy the database.
    - Run the binary. Ex.: `/path/to/binary`.
    - The app will deploy the database automatically the first time it is run.
    - If the database does not automatically deploy, use the `--deploy-db` flag to the binary.
    - Either way, you will see a success message.
    - Note that a default config file for the app will be created as well. Do not delete this!

1. Install & Configure NGINX.
    - NGINX is only used for HTTPS termination and proxying to the internal port.
    - `sudo apt install nginx`.
    - Test by running `nginx -v`.
    - See the "nginx-sample-https" file in "documentation/https_configuration/" for further steps.
    - Make sure the NGINX configuration points to the correct internal port noted in the app's logging output (in the "proxy_pass" directive).

1. Configure HTTPS certificate.
    - HTTPS should be used for security purposes. This is especially so if you plan on serving this app on the WAN.
    - See the "using-https-certificates" file in the documentation for further instructions.

1. Run the app.
    - Run the binary. Ex.: `/path/to/binary`.
    - You will see some logging output saying the app is running.
    - Access the app by running a web browser pointing to the IP or FQDN of the server the app is running on. This should be successful if you have your firewall and NGINX configuration correct.
    - You can log in using the default user, "admin@example.com", with the password being "admin@example.com".

1. Enable the app to run automatically.
    - This handles starting the app when the server boots.
    - See the "systemd-service" file in the documentation.
    - Logging is output to the systemd journal. See the "systemd-service" file for more details.
    - Copy the "systemd-service.txt" file to "my-app.service".
    - Edit the "my-app.service" file. Modify the ExecStart and User properties to the correct executable path and the user running the executable.
    - Move the "my-app.service" file to the `/etc/systemd/user` directory.
    - Enable the service running `systemctl enable /etc/systemd/user/my-app.service`.
    - Reboot the server to make sure the app starts when the server boots.

1. Change the default user password and disable the user.
    - The default user, admin@example.com, has a default password set. This is a security issue. Only use the default user to create your initial users.
    - Log in with the default admin@example.com username and password.
    - Create a new user with full permissions in the the app.
    - Log out and log back in as the new user.
    - Choose the admin@example.com user, change the password to something long and complex, mark the user as inactive, and turn off all permissions.
