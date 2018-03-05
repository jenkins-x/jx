pipeline {
    agent {
        label "jenkins-go"
    }
    environment {
        JOB_NAME            = "$JOB_NAME"
        BRANCH_NAME         = "$BRANCH_NAME"
        BUILD_NUMBER        = "$BUILD_NUMBER"
        GITHUB_ACCESS_TOKEN = "$GH_CREDS_PSW"
        GIT_API_TOKEN       = "$GH_CREDS_PSW"
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
            environment {
                GH_CREDS = credentials('jenkins-x-github')
            }
            when {
                branch 'master'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    container('go') {
                        sh "echo \$(jx-release-version) > pkg/version/VERSION"
                        sh "GITHUB_ACCESS_TOKEN=$GH_CREDS_PSW make release"
                    }
                }
            }
        }
    }
}
