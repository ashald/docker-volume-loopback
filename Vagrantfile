Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"
  config.vm.provision :docker
  config.vm.provision "shell", inline: "apt update && apt install -y jq xfsprogs"
end
