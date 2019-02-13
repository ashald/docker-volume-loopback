Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/bionic64"
  config.vm.provision :docker
  config.vm.provision "shell", inline: "apt-get update && apt-get install -y jq xfsprogs"
end
