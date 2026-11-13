#!/bin/sh
# Hash the tables that Q.6 must leave unchanged. Usage: q6-hashes.sh <store.sqlite>
# jobs is hashed without the two score columns (match_score, match_profile_revision), which may change.
DB="$1"
[ -f "$DB" ] || { echo "usage: $0 <store.sqlite>" >&2; exit 2; }
for t in user_profiles profile_documents profile_facts profile_revisions assessments assessment_queue queue_state application_drafts application_revisions workflow_history feedback; do
  h=$(sqlite3 "$DB" "select * from $t order by rowid" | shasum -a 256 | cut -d' ' -f1)
  echo "$t $(sqlite3 "$DB" "select count(*) from $t") $h"
done
cols=$(sqlite3 "$DB" "select group_concat(name, ',') from pragma_table_info('jobs') where name not in ('match_score','match_profile_revision')")
h=$(sqlite3 "$DB" "select $cols from jobs order by id" | shasum -a 256 | cut -d' ' -f1)
echo "jobs(without score columns) $(sqlite3 "$DB" "select count(*) from jobs") $h"
