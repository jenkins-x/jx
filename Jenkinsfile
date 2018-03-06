pipeline {
    agent {
        label "jenkins-go"
    }
    environment {
        GH_CREDS            = credentials('jenkins-x-github')
        BUILD_NUMBER        = "$BUILD_NUMBER"
        GIT_USERNAME        = "$GH_CREDS_USR"
        GIT_API_TOKEN       = "$GH_CREDS_PSW"
        GITHUB_ACCESS_TOKEN = "$GH_CREDS_PSW"

        JOB_NAME            = "$JOB_NAME"
        BRANCH_NAME         = "$BRANCH_NAME"
    }
    stages {
        stage('CI Build and Test') {
            when {
                branch 'PR-*'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    container('go') {
                        sh "make"
                        sh "make test"
                        sh "./build/jx --help"
                        sh "make preview"
                    }
                }
            }
        }
    
        stage('Build and Release') {
            when {
                branch 'master'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    container('go') {
                        sh "echo \$(jx-release-version) > pkg/version/VERSION"
                        sh "make release"
                    }
                }
            }
        }
    }
}
