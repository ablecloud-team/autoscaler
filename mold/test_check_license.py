# SPDX-License-Identifier: Apache-2.0
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
import copy
import pathlib
import tempfile
import unittest
import importlib.util
spec = importlib.util.spec_from_file_location("check_license", pathlib.Path(__file__).with_name("check-license.py"))
check_license = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check_license)

class CoverageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.temp.name).resolve()
        (self.root / 'LICENSE').write_text('Apache License, Version 2.0')
        (self.root / 'README.md').write_text('Existing project document')
        self.row = {'path': 'README.md', 'sha256': check_license.digest(self.root/'README.md'),
                    'licenseFile': 'LICENSE', 'licenseSha256': check_license.digest(self.root/'LICENSE'),
                    'reason': 'inherited-project-metadata'}
        self.ledger = {'files': [self.row], 'licenseNotices': {'LICENSE': self.row['licenseSha256']}}

    def tearDown(self):
        self.temp.cleanup()

    def test_document_inherits_exact_original_notice(self):
        self.assertEqual(check_license.verify_coverage(self.root,self.ledger), {'README.md'})

    def test_source_changed_or_notice_missing_fails_closed(self):
        (self.root/'README.md').write_text('changed')
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,self.ledger)
        (self.root/'README.md').write_text('Existing project document')
        (self.root/'LICENSE').unlink()
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,self.ledger)

    def test_firstparty_source_cannot_use_metadata_exception(self):
        (self.root/'source.go').write_text('package firstparty')
        self.row.update(path='source.go',sha256=check_license.digest(self.root/'source.go'))
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,self.ledger)

    def test_external_source_cannot_use_project_license_exception(self):
        self.row['reason']='preserved-third-party-notice'
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,self.ledger)

    def test_traversal_symlink_or_wildcard_are_rejected(self):
        for path in ('../README.md','/etc/passwd','*.md','README[1].md'):
            with self.subTest(path=path), self.assertRaises(ValueError):
                check_license.member(self.root,path) if '*' not in path and '[' not in path else check_license.verify_coverage(self.root,{'files':[{**self.row,'path':path}],'licenseNotices':self.ledger['licenseNotices']})
        (self.root/'link.md').symlink_to(self.root/'README.md')
        with self.assertRaises(ValueError):
            check_license.member(self.root,'link.md')

    def test_duplicate_coverage_and_notice_inventory_changes_are_rejected(self):
        ledger=copy.deepcopy(self.ledger)
        ledger['files'].append(copy.deepcopy(self.row))
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,ledger)
        self.ledger['licenseNotices']={}
        with self.assertRaises(ValueError):
            check_license.verify_coverage(self.root,self.ledger)

    def test_oci_exception_cannot_cover_other_provider_or_vendor(self):
        for path in ('cluster-autoscaler/cloudprovider/cloudstack/source.go','cluster-autoscaler/cloudprovider/oci/vendor-internal/source.go'):
            target=self.root/path;target.parent.mkdir(parents=True,exist_ok=True);target.write_text('package provider')
            self.row.update(path=path,sha256=check_license.digest(target),reason='upstream-oci-project-license')
            with self.assertRaises(ValueError):
                check_license.verify_coverage(self.root,self.ledger)

if __name__=='__main__':
    unittest.main()
