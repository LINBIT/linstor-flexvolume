Name: linstor-flexvolume
Version: 0.7.3
Release: 1%{?dist}
Summary: LINSTOR flexvolume plugin
License: GPLv2+
ExclusiveOS: linux
Source: %{name}-%{version}.tar.gz
Group: Applications/System

%define K8SPATH /usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~linstor-flexvolume
%define OPENSHIFTPATH /usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~linstor-flexvolume

%description
Flexvolume driver implementation for Linstor volumes


%prep
%setup -q

%build

%install
mkdir -p %{buildroot}/%{K8SPATH}
cp %{_builddir}/%{name}-%{version}/%{name} %{buildroot}/%{K8SPATH}/
mkdir -p %{buildroot}/%{OPENSHIFTPATH}
cp %{_builddir}/%{name}-%{version}/%{name} %{buildroot}/%{OPENSHIFTPATH}/

### kubernetes
%package kubernetes
Summary: Google's Container Orchestration Platform
Requires: linstor-satellite linstor-server

%description kubernetes
Kubernetes manages the lifecycle of containerized applications


%files kubernetes
%{K8SPATH}/%{name}

### openshift
%package openshift
Summary: Red Hat's Container Orchestration Platform
Requires: linstor-satellite linstor-server

%description openshift
Openshift manages the lifecycle of containerized applications and has a GUI


%files openshift
%{OPENSHIFTPATH}/%{name}

%changelog
* Tue Jul 31 2018 Roland Kammerer <roland.kammerer@linbit.com> 0.7.3-1
-  New upstream release
