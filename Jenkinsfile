pipeline {
    agent any

    environment {
        CHARTMUSEUM_CREDS   = credentials('jenkins-x-chartmuseum')
        GH_CREDS            = credentials('jx-pipeline-git-github-github')

        GIT_USERNAME        = "$GH_CREDS_USR"	
        GIT_API_TOKEN       = "$GH_CREDS_PSW"	
        GITHUB_ACCESS_TOKEN = "$GH_CREDS_PSW"

        JOB_NAME            = "$JOB_NAME"
        BRANCH_NAME         = "$BRANCH_NAME"
        ORG                 = 'jenkinsxio'
        APP_NAME            = 'jx'

    }
    options {
        skipDefaultCheckout(true)
    }
    stages {
        stage('Build and Release') {
            when {
                environment name: 'JOB_TYPE', value: 'postsubmit'
            }
            steps {
                dir ('/workspace') {

                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"
                    sh "echo \$(jx-release-version) > pkg/version/VERSION"
                    sh "make release"
                }
                dir ('/workspace/charts/jx') {

                    sh "git config --global credential.helper store"
                    sh "jx step git credentials"
                    sh "helm init --client-only"
                    sh "make release"
                }
                dir ('/workspace') {

                    sh "updatebot push-version --kind helm jx `cat pkg/version/VERSION`"
                }
            }
        }
    }
}
