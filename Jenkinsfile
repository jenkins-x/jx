pipeline {
    agent any
    environment {
        CHARTMUSEUM_CREDS   = credentials('jenkins-x-chartmuseum')
        JENKINS_CREDS       = credentials('test-jenkins-user')
        GH_CREDS            = credentials('jx-pipeline-git-github-github')
        GHE_CREDS           = credentials('jx-pipeline-git-github-ghe')
        GKE_SA              = credentials('gke-sa')

        GIT_USERNAME        = "$GH_CREDS_USR"	
        GIT_API_TOKEN       = "$GH_CREDS_PSW"	
        GITHUB_ACCESS_TOKEN = "$GH_CREDS_PSW"

        JOB_NAME            = "$JOB_NAME"
        BRANCH_NAME         = "$BRANCH_NAME"
        ORG                 = 'jenkinsxio'
        APP_NAME            = 'jx'
        PREVIEW_VERSION     = "0.0.0-SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER"
        TEAM                = "$BRANCH_NAME-$BUILD_NUMBER".toLowerCase()
        PREVIEW_IMAGE_TAG   = "SNAPSHOT-JX-$BRANCH_NAME-$BUILD_NUMBER"

        // for BDD tests
        GIT_PROVIDER_URL     = "https://github.beescloud.com"
        GHE_TOKEN            = "$GHE_CREDS_PSW"

        JX_DISABLE_DELETE_APP  = "true"
        JX_DISABLE_DELETE_REPO = "true"
    }
    stages {
        stage('CI Build and Test') {
            when {
                branch 'PR-*'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"

                    sh "echo building Pull Request for preview ${TEAM}"

                    sh "make linux"
                    sh 'test `git status --short | tee /dev/stderr | wc --bytes` -eq 0'
                    sh "make test-slow-integration"
                    sh "./build/linux/jx --help"

                    sh "docker build -t docker.io/$ORG/$APP_NAME:$PREVIEW_VERSION ."

                    sh "make preview"

                    // lets create a team for this PR and run the BDD tests
                    sh "gcloud auth activate-service-account --key-file $GKE_SA"
                    sh "gcloud container clusters get-credentials anthorse --zone europe-west1-b --project jenkinsx-dev"


                    sh "sed 's/\$VERSION/${PREVIEW_IMAGE_TAG}/g' myvalues.yaml.template > myvalues.yaml"
                    sh "echo the myvalues.yaml file is:"
                    sh "cat myvalues.yaml"

                    sh "echo creating team: ${TEAM}"

                    sh "git config --global --add user.name JenkinsXBot"
                    sh "git config --global --add user.email jenkins-x@googlegroups.com"

                    sh "cp ./build/linux/jx /usr/bin"

                    // lets trigger the BDD tests in a new team and git provider
                    sh "./build/linux/jx step bdd -b  --provider=gke --git-provider=ghe --git-provider-url=https://github.beescloud.com --git-username dev1 --git-api-token $GHE_CREDS_PSW --default-admin-password $JENKINS_CREDS_PSW --no-delete-app --no-delete-repo --tests install --tests test-create-spring"
                }
            }
        }

        stage('Build and Release') {
            when {
                branch 'master'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    git 'https://github.com/jenkins-x/jx'

                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"
                    sh "echo \$(jx-release-version) > pkg/version/VERSION"
                    sh "make release"
                }
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx/charts/jx') {

                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"
                    sh "helm init --client-only"
                    sh "make release"
                }
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm

                    sh "updatebot push-version --kind helm jx `cat pkg/version/VERSION`"
                }
            }
        }
    }
}
