# Microsoft Azure Container Networking
CNI Multitenacy binaries are meant only for 1st party customers for now.

Conflist Fields Description
---------------------------
multiTenancy - To indicate CNI to use multitenancy network setup using ovs bridge

enableExactMatchForPodName - This field is processed only incase if multitenancy is true.
                             If this set to false, then CNI strips the last two hex fields added by container runtime.
                             Eg: if container name is samplepod, then container runtime would
                             generate this as samplepod-3e4a-5e4a. CNI would strip 3e4a-5e4a and keep it as samplepod.
                             If the field is set to true, CNI would take whatever container runtime provides.

enableSnatOnHost - This field is processed only incase if multitenancy is true. If pod/container wants outbound connectivity,
                    this field should be set to true. Enabling this field also enables ip forwarding kernel setting in VM and adds
                    iptable rule to allow forward traffic from snat bridge.

