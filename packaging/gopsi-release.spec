Name:           gopsi-release
Version:        0.2.0
Release:        1%{?dist}
Summary:        Gopsi YUM repository configuration
License:        MIT
URL:            https://github.com/Venky0827/Gopsi

Source0:        gopsi.repo

%description
This package installs the YUM repository configuration for Gopsi.

%prep

%build

%install
install -Dm644 %{SOURCE0} %{buildroot}/etc/yum.repos.d/gopsi.repo

%files
/etc/yum.repos.d/gopsi.repo

%changelog
* Thu Nov 27 2025 Venky <anandbandari0@gmail.com> - 0.2.0-1
- Initial repo config package
