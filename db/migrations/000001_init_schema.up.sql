create table users (
    id bigint not null auto_increment primary key,
    name varchar(20) not null,
    display_name varchar(20) not null,
    handle varchar(20) not null,
    email varchar(255) not null,
    hashed_password varchar(255) not null,
    bio varchar(255),
    avatar_url varchar(2048) not null default 'https://cdn.example.com/avatars/default.png',
    birthday date,

    created_at datetime(6) not null default current_timestamp(6),
    updated_at datetime(6) null default null on update current_timestamp(6),

    unique key uq_users_handle (handle),
    unique key uq_users_email (email)
);

create table follows (
    follower_id bigint not null,
    following_id bigint not null,
    created_at datetime(6) not null default current_timestamp(6),

    primary key(follower_id, following_id),
    key idx_follows_following (following_id, follower_id),
    constraint fk_follows_follower foreign key (follower_id) references users(id) on delete cascade,
    constraint fk_follows_following foreign key (following_id) references users(id) on delete cascade,
    constraint chk_follows_different_users check (follower_id <> following_id)
);

create table language (
    id bigint not null auto_increment primary key,
    code varchar(20) not null,
    name varchar(255) not null,
    native_name varchar(255) not null,
    description varchar(255) not null,

    unique key uq_code (code)
);

create table user_setting (
    user_id bigint not null primary key,
    ui_mode varchar(20) not null default 'light',
    timezone varchar(20) not null default 'Asia/Tokyo',

    constraint fk_user_setting_id foreign key (user_id) references users(id) on delete cascade
);

create table user_language_setting (
    user_id bigint not null,
    language_code varchar(20) not null default 'ja',
    is_primary boolean not null default 0,

    primary key (user_id, language_code),
    constraint fk_user_language_setting_id foreign key (user_id) references users(id) on delete cascade,
    constraint fk_language_setting_code foreign key (language_code) references language(code)
);

create table user_notification_setting (
    id bigint not null auto_increment primary key,
    user_id bigint not null,
    channel varchar(20) not null,
    is_enabled boolean not null default 1,

    unique key uq_user_channel (user_id, channel),
    constraint fk_user_notification_setting_id foreign key (user_id) references users(id) on delete cascade
);

create table questionnaire (
    id bigint not null auto_increment primary key,
    created_by bigint not null,
    title varchar(100) not null,
    description varchar(255) not null,
    status varchar(20) not null,
    deadline datetime(6) not null,

    type varchar(20) not null,
    max_choices int not null,
    language_code varchar(20) not null,
    language_detected boolean not null default 0,

    visibility varchar(20) not null,

    created_at datetime(6) default current_timestamp(6),
    updated_at datetime(6) null default null on update current_timestamp(6),

    key idx_questionnaire_status_created (status, created_at),
    key idx_questionnaire_created_by (created_by, created_at),
    constraint fk_created_by foreign key (created_by) references users(id) on delete cascade,
    constraint fk_questionnaire_language_code foreign key (language_code) references language(code)
);

create table choice (
    id bigint not null auto_increment primary key,
    questionnaire_id bigint not null,
    title varchar(20) not null,
    display_order int not null,

    constraint fk_choice_questionnaire_id foreign key (questionnaire_id) references questionnaire(id) on delete cascade
);

create table vote (
    id bigint not null auto_increment primary key,
    questionnaire_id bigint not null,
    user_id bigint not null,
    created_at datetime(6) default current_timestamp(6),

    unique key uq_vote_questionnaire_user (questionnaire_id, user_id),
    constraint fk_vote_questionnaire_id foreign key (questionnaire_id) references questionnaire(id) on delete cascade,
    constraint fk_vote_user_id foreign key (user_id) references users(id) on delete cascade
);

create table answer_item (
    id bigint not null auto_increment primary key,
    vote_id bigint not null,
    choice_id bigint not null,

    unique key uq_answer_item_vote_choice (vote_id, choice_id),
    constraint fk_vote_id foreign key (vote_id) references vote(id) on delete cascade,
    constraint fk_choice_id foreign key (choice_id) references choice(id) on delete cascade
);

create table comments (
    id bigint not null auto_increment primary key,
    questionnaire_id bigint not null,
    user_id bigint not null,
    content varchar(255) not null,
    parent_comment_id bigint,
    created_at datetime(6) default current_timestamp(6),

    key idx_comments_questionnaire_created (questionnaire_id, created_at),
    constraint fk_comments_questionnaire_id foreign key (questionnaire_id) references questionnaire(id) on delete cascade,
    constraint fk_comments_user_id foreign key (user_id) references users(id) on delete cascade,
    constraint fk_parent_comment_id foreign key (parent_comment_id) references comments(id) on delete cascade
);

create table questionnaire_like (
    questionnaire_id bigint not null,
    user_id bigint not null,
    created_at datetime(6) default current_timestamp(6),

    primary key (questionnaire_id, user_id),
    constraint fk_questionnaire_like_id foreign key (questionnaire_id) references questionnaire(id) on delete cascade,
    constraint fk_questionnaire_like_user_id foreign key (user_id) references users(id) on delete cascade
);
