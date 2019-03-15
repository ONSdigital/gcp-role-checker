import argparse
import json
import os
import subprocess
import sys

top_path = os.path.dirname(os.path.dirname(os.path.realpath(__file__)))

parser = argparse.ArgumentParser(description='''
Script to generate a list of service accounts along with their privilege levels
''')
parser.add_argument('org',
                    help='The organization resource ID I.e. organizations/999999999999')
parser.add_argument('--project_labels',
                    help='A set of labels to filter projects on \n' +
                        'I.e. env:dev,project:foo')

parser.add_argument('--data_dir', help='location of raw JSON data', default='data')
parser.add_argument('--limit', default=10, type=int,
                    help='the max number of accounts to return')
parser.add_argument('--member_type',
                    choices=['service_account','user_account','group'],
                    help='the type of member to filter results by')
parser.add_argument('--skip_collect', action='store_true',
                    help='gather data from Google APIs')
parser.add_argument('--sort_type', default='total_sum',
                    choices=['total_sum', 'top_sum'],
                    help='the sort function used to order the members')


def main():
    parsed_args = parser.parse_args()

    if not parsed_args.skip_collect:
        gather_data(parsed_args.org, parsed_args.project_labels, parsed_args.data_dir)

    with open(os.path.join(parsed_args.data_dir, 'members.json')) as f:
        members = json.load(f)

    service_accounts = filter_service_accounts(members, parsed_args.member_type)

    sort_fn = getattr(sys.modules[__name__], parsed_args.sort_type)
    sorted_members = sorted(service_accounts, key=sort_fn, reverse=True)
    print_permissions(sorted_members[:parsed_args.limit])


def print_permissions(permissions):
    for item in permissions:
        email, data = item
        print("\n{}:".format(email))
        for resource in data['resources']:
            roles = [
                "{} ({} permissions)".format(role['name'], role['permission_count'])
                for role in resource['roles']
            ]
            print("    {}: {}".format(resource['name'], ",".join(roles)))


def gather_data(org, project_labels, data_dir):
    proc_args = [
        "go",
        "run",
        f'{top_path}/cmd/checker/main.go',
        f'-org={org}',
    ]
    if project_labels:
        proc_args.append(f'-project_labels={project_labels}')
    if data_dir:
        proc_args.append(f'-data={data_dir}')
    subprocess.run(proc_args, check=True)


def filter_service_accounts(members, member_type):
    startswith_map = {
        'service_account': 'serviceAccount:',
        'user_account': 'user:',
        'group': 'group:',
        None: ''
    }
    return (
        (memberEmail, member) for memberEmail, member in members.items()
        if memberEmail.startswith(startswith_map[member_type])
    )


def total_sum(data):
    _, member = data
    return sum(
        sum(
            role['permission_count'] for role in resource['roles']
        )
        for resource in member['resources']
    )


def top_sum(data):
    _, member = data
    return max(
        sum(
            role['permission_count'] for role in resource['roles']
        )
        for resource in member['resources']
    )


if __name__ == '__main__':
    main()