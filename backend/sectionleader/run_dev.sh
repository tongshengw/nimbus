#!/bin/bash
rm -f server.log
sudo rm -rf /home/tswu/frpc/nimbus
sudo mkdir /home/tswu/frpc/nimbus
# sudo rm -rf _data/
sudo rm -rf /var/lib/cni/networks
sudo rm -rf /etc/cni/conf.d

for iface in $(ip -o link show | awk -F': ' '{print $2}' | cut -d@ -f1 | grep 'veth'); do
    sudo ip link delete "$iface"
done

make
#sudo ./server-sectionleader