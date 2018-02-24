#!/bin/bash

gitRepositoryPath=testing-repository

cd $gitRepositoryPath || exit 1

# Create branch test
git checkout --quiet -b test

# Create several commits on test branch
touch file1
git add file1
git commit --quiet -F- <<EOF
feat(file1) : new file 1

create a new file 1
EOF

touch file2
git add file2
git commit --quiet -F- <<EOF
feat(file2) : new file 2

create a new file 2
EOF

# Create branch test1
git checkout --quiet -b test1

touch file3
git add file3
git commit --quiet -F- <<EOF
feat(file3) : new file 3

create a new file 3
EOF

touch file4
git add file4
git commit --quiet -F- <<EOF
feat(file4) : new file 4

create a new file 4
EOF

# Create branch test2
git checkout --quiet -b test2

touch file5
git add file5
git commit --quiet -F- <<EOF
feat(file5) : new file 5

create a new file 5
EOF

touch file6
git add file6
git commit --quiet -F- <<EOF
feat(file6) : new file 6

create a new file 6
EOF

# Checkout branch test1
git checkout --quiet test1

# Merge branch test2
git merge --quiet --no-edit --no-ff test2

# Checkout branch test
git checkout --quiet test

# Merge branch test1
git merge --quiet --no-edit --no-ff test1

# Create several commits on main test branch

touch file7
git add file7
git commit --quiet -F- <<EOF
feat(file7) : new file 7

create a new file 7
EOF

touch file8
git add file8
git commit --quiet -F- <<EOF
feat(file8) : new file 8

create a new file 8
EOF
