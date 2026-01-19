# nimbus
![1752532909494](https://github.com/user-attachments/assets/518f087a-a1bf-4e7d-8679-2cb0e07e4858)

This is a self-hostable virtual machine manager, with functionality similar to AWS EC2. It uses a public reverse proxy to allow hosting at home! Firecracker for lightweight virtual machine management and CNI for networking.

## TODO:

- [x] frpc setup
- [x] api to control vms
- [ ] switch to jailer
- [ ] port forwarding
- [ ] change auth to work with frontends/wrappers
- [ ] add a db
- [ ] testing newly provisioned machines accessible
- [ ] MINECRAFT SERVER
- [x] frontend website
- [ ] pool to instantly provision

## bugs:
- [ ] Incorrect shutdown, something to do with signals. I don't think this is fixable, since it is likely caused by interrupts being sent to firecracker process as well as sectionleader. Workaround for now is shutdown-all endpoint, and can delete veth network interfaces to fix.
