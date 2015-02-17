Vagrant.configure('2') do |config|
  config.vm.box = "ubuntu/trusty64" # Ubuntu 14.04
  config.vm.provision :shell, :privileged => false, :path => "scripts/install-go.sh"
  config.vm.provision :shell, :privileged => false, :path => "scripts/install-rocket.sh"
end
