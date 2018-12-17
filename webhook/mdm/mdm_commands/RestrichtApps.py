import sys
import os
#sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.insert(1, os.path.join(sys.path[0], '../..'))
# import ../../config.py
import config

from MDMCommand import MDMCommand

class RestrictApps( MDMCommand ):
    def __init__(self, udid, profile_path = './profiles/restrict_apps.mobileconfig'):
        """
        Parameters
        ------
        udid : str
            udid of the device (40 characters long) and lower cased
        profile_path: str, optional
            Path to the profile to be installed
        """
        self.profile_path = profile_path
        super(RestrictApps, self).__init__(udid)

    def _request_url(self):
        return config.MDM.server_url + "/v1/commands"

    def _request_type(self):
        return "InstallProfile"

    def _request_data(self):
        payload = self._profile_payload(self.profile_path)
        return {
            "udid": self.udid,
            "request_type": self._request_type(),
            "payload": payload,
        }

    def _command_identifier(self):
        return "install_profile_restrict_apps"

    def _command_id(self):
        return 2